package budgets

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// The history kinds of this package (014 plan internal/budgets, data model §1.4–§1.5, §3).
const (
	kindBudget             = "globalBudget" // top level
	kindBudgetSingular     = "global budget"
	kindBudgetPlural       = "global budgets"
	kindAllocation         = "budgetAllocation"  // a child of the plan: a plan's delete, Undo and restore carry it
	kindAllocationSingular = "budget allocation" // mid-sentence, as in "Not saved — Carlos Mendes changed this budget allocation"
	kindAllocationPlural   = "budget allocations"
	kindPlan               = "plan" // feature 004's kind, the allocations' parent
)

// allocationField starts the field of a global budget's log change about one allocation: "allocations.P-1001".
const allocationField = "allocations."

// The field labels of a global budget, as the activity log shows them.
const (
	labelName        = "Name"
	labelYear        = "Year"
	labelAmount      = "Amount"
	labelOwner       = "Owner"
	labelDescription = "Description"
	labelAllocation  = "Allocation to '%s'"
	textNotAllocated = "Not allocated"
)

// Kinds returns the history kinds of global budgets and allocations, for internal/app to register after the
// plan kind (allocations are children of plans). With them, history logs a global budget's changes in words,
// moves a global budget to the recycle bin and brings it back, and carries a plan's allocation with the plan.
func (s *Service) Kinds() []history.Kind {
	return []history.Kind{
		{
			Name: kindBudget, Singular: kindBudgetSingular, Plural: kindBudgetPlural,
			Snapshot: s.budgetSnapshot, Remove: s.repo.DeleteBudget, Put: s.putBudget,
			Describe: s.describe, PrepareRestore: s.prepareBudgetRestore,
		},
		{
			Name: kindAllocation, Singular: kindAllocationSingular, Plural: kindAllocationPlural, Parent: kindPlan,
			Snapshot: s.allocationSnapshot, ChildIDs: s.allocationIDsOfPlan, Remove: s.repo.DeleteAllocation,
			Put: s.putAllocation, PrepareRestore: s.prepareAllocationRestore,
		},
	}
}

// budgetSnapshot reads the global budget id as history keeps it; a global budget has no parent.
func (s *Service) budgetSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	b, found, err := s.repo.GetBudget(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	data, err := json.Marshal(b)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: b.ID, Name: b.Name, Data: data}, true, nil
}

// putBudget stores the global budget of a snapshot again, exactly as the snapshot holds it.
func (s *Service) putBudget(ctx context.Context, snapshot history.Snapshot) error {
	var b GlobalBudget
	if err := json.Unmarshal(snapshot.Data, &b); err != nil {
		return err
	}
	return s.repo.PutBudget(ctx, b)
}

// describe gives the activity log the label of a global budget's field and the text of its value (data
// model §3): money as "2,500,000.00 USD", the owner's name, and for an allocation the plan's name in the label
// and "Not allocated" when there is none. Anything else gets history's default.
func (s *Service) describe(ctx context.Context, field string, value any) (label, text string, err error) {
	if planID, ok := strings.CutPrefix(field, allocationField); ok {
		plan, found, err := s.plans.Plan(ctx, planID)
		if err != nil {
			return "", "", err
		}
		name := planID // a plan that no longer resolves is named by its id
		if found {
			name = plan.Name
		}
		return fmt.Sprintf(labelAllocation, name), cmp.Or(moneyText(value), textNotAllocated), nil
	}
	stored, _ := value.(string)
	switch field {
	case "name":
		return labelName, "", nil
	case "year":
		return labelYear, "", nil
	case "amount":
		if amount, err := money.ParseAmount(stored); err == nil {
			return labelAmount, money.USD(amount).Format(), nil
		}
		return labelAmount, "", nil
	case "ownerId":
		refs, err := s.directory.PersonRefs(ctx, []string{stored})
		return labelOwner, refs[stored].Name, err
	case "description":
		return labelDescription, "", nil
	}
	return "", "", nil
}

// moneyText writes a stored money value ({"amount": "1200000.00", "currency": "USD"}) as people read it,
// "1,200,000.00 USD"; "" for anything else.
func moneyText(value any) string {
	object, _ := value.(map[string]any)
	text, _ := object["amount"].(string)
	currency, _ := object["currency"].(string)
	amount, err := money.ParseAmount(text)
	if err != nil || currency == "" {
		return ""
	}
	return money.Money{Amount: amount, Currency: currency}.Format()
}

// prepareBudgetRestore applies the name rule just before a global budget comes back from the recycle bin
// (001's rule, as plans): an Undo of a global budget whose name another one has meanwhile taken is refused,
// and a Restore renames it "<name> (restored)", "<name> (restored 2)", …
func (s *Service) prepareBudgetRestore(ctx context.Context, rc history.RestoreContext, snapshot *history.Snapshot) ([]string, error) {
	var b GlobalBudget
	if err := json.Unmarshal(snapshot.Data, &b); err != nil {
		return nil, err
	}
	all, err := s.repo.ListBudgets(ctx)
	if err != nil {
		return nil, err
	}
	other, taken := budgetNamed(all, b.Name, b.ID)
	if !taken {
		return nil, nil
	}
	if rc.Undo {
		return nil, errs.NameTaken(fmt.Sprintf(undoNameTakenMessage, other.Name))
	}
	b.Name = restoredName(all, b.Name, b.ID)
	b.Touch(rc.Actor, rc.Now) // a changed global budget gets a new version
	data, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	snapshot.Name, snapshot.Data = b.Name, data
	return []string{fmt.Sprintf(renamedNote, b.Name)}, nil
}

// restoredName returns the first of "<name> (restored)", "<name> (restored 2)", … that no global budget among
// all other than id has.
func restoredName(all []GlobalBudget, name, id string) string {
	candidate := name + " (restored)"
	for n := 2; ; n++ {
		if _, taken := budgetNamed(all, candidate, id); !taken {
			return candidate
		}
		candidate = fmt.Sprintf("%s (restored %d)", name, n)
	}
}

// allocationSnapshot reads the allocation id as history keeps it: its parent is its plan, and its name is
// its global budget's.
func (s *Service) allocationSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	allocation, found, err := s.repo.GetAllocation(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	name := allocation.GlobalBudgetID
	if b, found, err := s.repo.GetBudget(ctx, allocation.GlobalBudgetID); err != nil {
		return history.Snapshot{}, false, err
	} else if found {
		name = b.Name
	}
	data, err := json.Marshal(allocation)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: allocation.ID, Name: name, ParentID: allocation.PlanID, Data: data}, true, nil
}

// allocationIDsOfPlan lists the allocation of the plan planID (at most one), so that a plan's delete takes it
// to the recycle bin (FR-035).
func (s *Service) allocationIDsOfPlan(ctx context.Context, planID string) ([]string, error) {
	allocation, found, err := s.repo.AllocationOfPlan(ctx, planID)
	if err != nil || !found {
		return nil, err
	}
	return []string{allocation.ID}, nil
}

// putAllocation stores the allocation of a snapshot again, exactly as the snapshot holds it, unless
// prepareAllocationRestore dropped it (Data null).
func (s *Service) putAllocation(ctx context.Context, snapshot history.Snapshot) error {
	if dropped(snapshot) {
		return nil
	}
	var allocation Allocation
	if err := json.Unmarshal(snapshot.Data, &allocation); err != nil {
		return err
	}
	return s.repo.PutAllocation(ctx, allocation)
}

// dropped reports whether prepareAllocationRestore left the allocation of snapshot out of a restore.
func dropped(snapshot history.Snapshot) bool {
	return len(snapshot.Data) == 0 || string(snapshot.Data) == "null"
}

// prepareAllocationRestore runs when a plan comes back from the recycle bin with its allocation (FR-035):
// the allocation comes back too, unless its global budget no longer exists (message 31) or the plan has
// meanwhile got another allocation; then it is left out, with a note, and the plan keeps its budget, which is
// editable again (as FR-034).
func (s *Service) prepareAllocationRestore(ctx context.Context, rc history.RestoreContext, snapshot *history.Snapshot) ([]string, error) {
	var allocation Allocation
	if err := json.Unmarshal(snapshot.Data, &allocation); err != nil {
		return nil, err
	}
	_, budgetFound, err := s.repo.GetBudget(ctx, allocation.GlobalBudgetID)
	if err != nil {
		return nil, err
	}
	other, taken, err := s.repo.AllocationOfPlan(ctx, allocation.PlanID)
	if err != nil {
		return nil, err
	}
	switch {
	case !budgetFound:
		snapshot.Data = nil
		return []string{noteBudgetDeleted}, nil
	case taken && other.ID != allocation.ID:
		snapshot.Data = nil
		return []string{notePlanReallocated}, nil
	}
	return nil, nil
}
