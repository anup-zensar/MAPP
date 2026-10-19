package settings

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/record"
)

// The limits of finance budgets and their items (data model §7, §8).
const (
	maxBudgetName = 120
	maxItemName   = 120
	maxItemCode   = 40
)

// Messages of the plan's "User-visible messages" table about finance budgets.
const (
	budgetGoneMessage      = "This finance budget no longer exists — someone may have deleted it."
	itemGoneMessage        = "This finance budget item no longer exists — someone may have deleted it."
	yearLockedMessage      = "The year can't change once the item is saved — create a new item for another year."
	chooseYearMessage      = "Choose a year from %d to %d."
	enterBudgetNameMessage = "Enter a budget name."
	itemNameTakenMessage   = "The budget '%s' already has an item called '%s'."
	itemCodeTakenMessage   = "The code '%s' is already used in %d by '%s'."
	budgetNameTakenMessage = "%d already has a budget called '%s'."
)

// The budget years that can be chosen: the current year − 1 to + 5 (FR-014).
const (
	yearsBefore = 1
	yearsAfter  = 5
)

// ListFinanceBudgets returns the finance budgets of year (the clock's UTC year when nil) that have at least
// one item, by name, each with its items by name, its active item count and its usage; and the years that
// can be chosen (FR-014, FR-015, US5-3). Every signed-in person may see them (FR-008).
func (s *Service) ListFinanceBudgets(ctx context.Context, a actor.Actor, year *int) (FinanceBudgetPage, error) {
	current := s.now().UTC().Year()
	shown := current
	if year != nil {
		shown = *year
	}
	page := FinanceBudgetPage{Items: []FinanceBudgetView{}, Year: shown}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		budgets, err := s.repo.ListFinanceBudgets(ctx)
		if err != nil {
			return err
		}
		items, err := s.repo.ListFinanceBudgetItems(ctx)
		if err != nil {
			return err
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		itemsOf := map[string][]FinanceBudgetItem{}
		for _, i := range items {
			itemsOf[i.BudgetID] = append(itemsOf[i.BudgetID], i)
		}
		years := map[int]bool{}
		for y := current - yearsBefore; y <= current+yearsAfter; y++ {
			years[y] = true
		}
		slices.SortStableFunc(budgets, func(x, y FinanceBudget) int { return compareNames(x.Name, y.Name) })
		for _, b := range budgets {
			if len(itemsOf[b.ID]) == 0 {
				continue
			}
			years[b.Year] = true
			if b.Year == shown {
				page.Items = append(page.Items, budgetView(b, itemsOf[b.ID], u))
			}
		}
		for y := range years {
			page.Years = append(page.Years, y)
		}
		slices.Sort(page.Years)
		return nil
	})
	page.Total = len(page.Items)
	return page, err
}

// GetFinanceBudgetItem returns the finance budget item id with its year, budget name, usage and locks
// (US5-4).
func (s *Service) GetFinanceBudgetItem(ctx context.Context, a actor.Actor, id string) (FinanceBudgetItemView, error) {
	var view FinanceBudgetItemView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		item, err := s.storedItem(ctx, id)
		if err != nil {
			return err
		}
		view, err = s.itemViewOf(ctx, item)
		return err
	})
	return view, err
}

// CreateFinanceBudgetItem creates an active finance budget item (FR-014, US5-3): a budget year from the
// current year − 1 to + 5, a budget name and an item name of at most 120 characters each, the item name
// free in its budget, and an optional code of at most 40 characters that no other item of the year has,
// ignoring letter case. The item joins the budget of that year with the same name, ignoring letter case,
// or a new budget is created with it: one "created" entry for the budget, then one for the item. Every
// problem is reported at once.
func (s *Service) CreateFinanceBudgetItem(ctx context.Context, a actor.Actor, input ItemInput) (FinanceBudgetItemView, error) {
	var view FinanceBudgetItemView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		var problems errs.Problems
		current := s.now().UTC().Year()
		year := 0
		if input.Year == nil || *input.Year < current-yearsBefore || *input.Year > current+yearsAfter {
			problems.Add("year", fmt.Sprintf(chooseYearMessage, current-yearsBefore, current+yearsAfter))
		} else {
			year = *input.Year
		}
		budgetName := checkBudgetName(&problems, "budgetName", input.BudgetName)
		item := FinanceBudgetItem{Name: strings.TrimSpace(input.Name), Code: strings.TrimSpace(input.Code)}
		budget, err := s.placeItem(ctx, item, year, budgetName, &problems)
		if err != nil {
			return err
		}
		if err := problems.Err(); err != nil {
			return err
		}
		if budget, err = s.saveNewBudget(ctx, a, budget); err != nil {
			return err
		}
		if item.ID, err = s.ids.Next(ctx, ids.PrefixFinanceBudgetItem); err != nil {
			return err
		}
		item.BudgetID, item.Active, item.Meta = budget.ID, true, record.NewMeta(a, s.now())
		if err := s.repo.PutFinanceBudgetItem(ctx, item); err != nil {
			return err
		}
		if err := s.logCreated(ctx, a, itemKind, item.ID, item.Name, ""); err != nil {
			return err
		}
		view, err = s.itemViewOf(ctx, item)
		return err
	})
	return view, err
}

// UpdateFinanceBudgetItem changes the name, the code or the budget of the finance budget item id (FR-014,
// US5-4, D-203), from the version the person started at (FR-007), with the rules of a new item. Its year
// never changes: a new budget name moves it to the budget of the same year with that name, ignoring letter
// case, or to a new budget of that year (logged as created first). A save that changes nothing writes no
// version and no log entry; otherwise one "changed" entry.
func (s *Service) UpdateFinanceBudgetItem(ctx context.Context, a actor.Actor, id string, update ItemUpdate) (FinanceBudgetItemView, error) {
	var view FinanceBudgetItemView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, err := s.storedItem(ctx, id)
		if err != nil {
			return err
		}
		current, _, err := s.repo.GetFinanceBudget(ctx, stored.BudgetID)
		if err != nil {
			return err
		}
		var problems errs.Problems
		checkVersion(&problems, update.Version)
		budgetName := current.Name
		if update.BudgetName != nil {
			budgetName = checkBudgetName(&problems, "budgetName", *update.BudgetName)
		}
		changed := stored
		if update.Name != nil {
			changed.Name = strings.TrimSpace(*update.Name)
		}
		if update.Code != nil {
			changed.Code = strings.TrimSpace(*update.Code)
		}
		budget, err := s.placeItem(ctx, changed, current.Year, budgetName, &problems)
		if err != nil {
			return err
		}
		if err := problems.Err(); err != nil {
			return err
		}
		if err := stored.CheckVersion(*update.Version, itemNoun); err != nil {
			return err
		}
		if budget, err = s.saveNewBudget(ctx, a, budget); err != nil {
			return err
		}
		changed.BudgetID = budget.ID
		changes, err := history.Diff(stored, changed)
		if err != nil {
			return err
		}
		if len(changes) > 0 {
			changed.Touch(a, s.now())
			if err := s.repo.PutFinanceBudgetItem(ctx, changed); err != nil {
				return err
			}
			if err := s.logChanged(ctx, a, itemKind, changed.ID, changed.Name, changes, ""); err != nil {
				return err
			}
			stored = changed
		}
		view, err = s.itemViewOf(ctx, stored)
		return err
	})
	return view, err
}

// RenameFinanceBudget renames the finance budget id (FR-015, US5-4): a name of at most 120 characters that
// no other budget of its year has, ignoring letter case. Items and plans refer to the budget by id, so all
// its items and the plans using it as their default show the new name at once. One "changed" entry on the
// budget; a rename to the same name changes nothing.
func (s *Service) RenameFinanceBudget(ctx context.Context, a actor.Actor, id string, input RenameInput) (FinanceBudgetView, error) {
	var view FinanceBudgetView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		b, found, err := s.repo.GetFinanceBudget(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(budgetGoneMessage)
		}
		var problems errs.Problems
		checkVersion(&problems, input.Version)
		name := checkBudgetName(&problems, "name", input.Name)
		if name != "" {
			other, taken, err := s.budgetNamed(ctx, b.Year, name, b.ID)
			if err != nil {
				return err
			}
			if taken {
				problems.Add("name", fmt.Sprintf(budgetNameTakenMessage, b.Year, other.Name))
			}
		}
		if err := problems.Err(); err != nil {
			return err
		}
		if err := b.CheckVersion(*input.Version, budgetNoun); err != nil {
			return err
		}
		changed := b
		changed.Name = name
		changes, err := history.Diff(b, changed)
		if err != nil {
			return err
		}
		if len(changes) > 0 {
			changed.Touch(a, s.now())
			if err := s.repo.PutFinanceBudget(ctx, changed); err != nil {
				return err
			}
			if err := s.logChanged(ctx, a, budgetKind, changed.ID, changed.Name, changes, ""); err != nil {
				return err
			}
			b = changed
		}
		items, err := s.repo.ItemsOf(ctx, b.ID)
		if err != nil {
			return err
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		view = budgetView(b, items, u)
		return nil
	})
	return view, err
}

// SetFinanceBudgetItemActive sets the finance budget item id inactive or reactivates it (FR-004): line
// items cannot link to an inactive item; those linked keep it. The same status again changes nothing.
func (s *Service) SetFinanceBudgetItemActive(ctx context.Context, a actor.Actor, id string, update StatusUpdate) (FinanceBudgetItemView, error) {
	active, version, err := checkStatus(update)
	if err != nil {
		return FinanceBudgetItemView{}, err
	}
	var view FinanceBudgetItemView
	err = s.tx.Write(ctx, func(ctx context.Context) error {
		item, err := s.storedItem(ctx, id)
		if err != nil {
			return err
		}
		if err := item.CheckVersion(version, itemNoun); err != nil {
			return err
		}
		if item.Active != active {
			item.Active = active
			item.Touch(a, s.now())
			if err := s.repo.PutFinanceBudgetItem(ctx, item); err != nil {
				return err
			}
			if err := s.logStatus(ctx, a, itemKind, item.ID, item.Name, active); err != nil {
				return err
			}
		}
		view, err = s.itemViewOf(ctx, item)
		return err
	})
	return view, err
}

// DeleteFinanceBudgetItem moves the finance budget item id to the recycle bin, under its budget, so that it
// can be undone (FR-004, US5-5). An item linked to line items — also those in the recycle bin — is refused
// with in_use: set it inactive instead. Its budget stays (D-047).
func (s *Service) DeleteFinanceBudgetItem(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		item, err := s.storedItem(ctx, id)
		if err != nil {
			return err
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		if usage := u.item(item.ID); usage.InUse {
			return inUseError(item.Name, usage)
		}
		entry, err = s.history.MoveToBin(ctx, a, itemKind, item.ID)
		return err
	})
	return entry, err
}

// storedItem reads the finance budget item id, or answers not found.
func (s *Service) storedItem(ctx context.Context, id string) (FinanceBudgetItem, error) {
	item, found, err := s.repo.GetFinanceBudgetItem(ctx, id)
	if err != nil {
		return FinanceBudgetItem{}, err
	}
	if !found {
		return FinanceBudgetItem{}, errs.NotFound(itemGoneMessage)
	}
	return item, nil
}

// checkBudgetName trims a budget name and adds its problem to problems on field: empty, or longer than 120
// characters. It returns the trimmed name, or "" when it has a problem.
func checkBudgetName(problems *errs.Problems, field, name string) string {
	name = strings.TrimSpace(name)
	checkText(problems, field, name, enterBudgetNameMessage, maxBudgetName)
	if utf8.RuneCountInString(name) > maxBudgetName {
		return ""
	}
	return name
}

// placeItem finds the budget the item goes into — the budget of year named budgetName, ignoring letter
// case, or a new one (no id yet) — and adds the problems of the item's trimmed name and code there to
// problems (FR-014): a name of 1–120 characters that no other item of the budget has, and a code of at most
// 40 characters that no other item of the year has, both ignoring letter case. A year of 0 or an empty
// budget name had a problem already: then only the item's own fields are checked.
func (s *Service) placeItem(ctx context.Context, item FinanceBudgetItem, year int, budgetName string, problems *errs.Problems) (FinanceBudget, error) {
	budget := FinanceBudget{Year: year, Name: budgetName}
	known := year != 0 && budgetName != ""
	if known {
		found, taken, err := s.budgetNamed(ctx, year, budgetName, "")
		if err != nil {
			return FinanceBudget{}, err
		}
		if taken {
			budget = found
		}
	}

	checkText(problems, "name", item.Name, enterNameMessage, maxItemName)
	if known && budget.ID != "" && item.Name != "" && utf8.RuneCountInString(item.Name) <= maxItemName {
		siblings, err := s.repo.ItemsOf(ctx, budget.ID)
		if err != nil {
			return FinanceBudget{}, err
		}
		if other, taken := findNamed(siblings, item.Name, item.ID, func(i FinanceBudgetItem) (string, string) { return i.ID, i.Name }); taken {
			problems.Add("name", fmt.Sprintf(itemNameTakenMessage, budget.Name, other.Name))
		}
	}

	checkLength(problems, "code", item.Code, maxItemCode)
	if year != 0 && item.Code != "" && utf8.RuneCountInString(item.Code) <= maxItemCode {
		other, taken, err := s.itemWithCodeInYear(ctx, year, item.Code, item.ID)
		if err != nil {
			return FinanceBudget{}, err
		}
		if taken {
			problems.Add("code", fmt.Sprintf(itemCodeTakenMessage, item.Code, year, other.Name))
		}
	}
	return budget, nil
}

// saveNewBudget stores the budget b when it is new (no id yet), with one "created" entry, and returns it
// with its id; a stored budget is returned as it is (data model §7: a budget is created with its first
// item).
func (s *Service) saveNewBudget(ctx context.Context, a actor.Actor, b FinanceBudget) (FinanceBudget, error) {
	if b.ID != "" {
		return b, nil
	}
	id, err := s.ids.Next(ctx, ids.PrefixFinanceBudget)
	if err != nil {
		return FinanceBudget{}, err
	}
	b.ID, b.Meta = id, record.NewMeta(a, s.now())
	if err := s.repo.PutFinanceBudget(ctx, b); err != nil {
		return FinanceBudget{}, err
	}
	return b, s.logCreated(ctx, a, budgetKind, b.ID, b.Name, "")
}

// budgetNamed returns the budget of year, other than exceptID, named name ignoring letter case and
// surrounding spaces, and false when there is none.
func (s *Service) budgetNamed(ctx context.Context, year int, name, exceptID string) (FinanceBudget, bool, error) {
	budgets, err := s.repo.ListFinanceBudgets(ctx)
	if err != nil {
		return FinanceBudget{}, false, err
	}
	sameYear := filter(budgets, func(b FinanceBudget) bool { return b.Year == year })
	b, found := findNamed(sameYear, name, exceptID, func(b FinanceBudget) (string, string) { return b.ID, b.Name })
	return b, found, nil
}

// itemWithCode returns the item, other than exceptID, with the code code ignoring letter case in the year
// of the budget budgetID, and false when there is none (FR-014).
func (s *Service) itemWithCode(ctx context.Context, budgetID, code, exceptID string) (FinanceBudgetItem, bool, error) {
	budget, found, err := s.repo.GetFinanceBudget(ctx, budgetID)
	if err != nil || !found {
		return FinanceBudgetItem{}, false, err
	}
	return s.itemWithCodeInYear(ctx, budget.Year, code, exceptID)
}

// itemWithCodeInYear returns the item, other than exceptID, with the code code ignoring letter case among
// the items of the budgets of year, and false when there is none (FR-014).
func (s *Service) itemWithCodeInYear(ctx context.Context, year int, code, exceptID string) (FinanceBudgetItem, bool, error) {
	budgets, err := s.repo.ListFinanceBudgets(ctx)
	if err != nil {
		return FinanceBudgetItem{}, false, err
	}
	items, err := s.repo.ListFinanceBudgetItems(ctx)
	if err != nil {
		return FinanceBudgetItem{}, false, err
	}
	sameYear := map[string]bool{}
	for _, b := range budgets {
		sameYear[b.ID] = b.Year == year
	}
	for _, i := range items {
		if i.ID != exceptID && sameYear[i.BudgetID] && i.Code != "" && sameName(i.Code, code) {
			return i, true, nil
		}
	}
	return FinanceBudgetItem{}, false, nil
}

// itemViewOf returns the item as the API answers it, reading its budget and the usage index.
func (s *Service) itemViewOf(ctx context.Context, item FinanceBudgetItem) (FinanceBudgetItemView, error) {
	budget, _, err := s.repo.GetFinanceBudget(ctx, item.BudgetID)
	if err != nil {
		return FinanceBudgetItemView{}, err
	}
	u, err := s.usage(ctx)
	if err != nil {
		return FinanceBudgetItemView{}, err
	}
	return itemView(item, budget, u), nil
}

// budgetView returns the budget b with its items as the API answers it.
func budgetView(b FinanceBudget, items []FinanceBudgetItem, u *usageIndex) FinanceBudgetView {
	view := FinanceBudgetView{ID: b.ID, Year: b.Year, Name: b.Name, Items: []FinanceBudgetItemView{}, Usage: u.budget(b.ID), Meta: b.Meta}
	sorted := slices.Clone(items)
	slices.SortStableFunc(sorted, func(x, y FinanceBudgetItem) int { return compareNames(x.Name, y.Name) })
	for _, i := range sorted {
		view.Items = append(view.Items, itemView(i, b, u))
		if i.Active {
			view.ActiveItemCount++
		}
	}
	return view
}

// itemView returns the item i of the budget b as the API answers it. Its year never changes (data model §8).
func itemView(i FinanceBudgetItem, b FinanceBudget, u *usageIndex) FinanceBudgetItemView {
	return FinanceBudgetItemView{
		ID: i.ID, BudgetID: i.BudgetID, BudgetName: b.Name, Year: b.Year, Name: i.Name, Code: i.Code, Active: i.Active,
		Usage: u.item(i.ID), Locks: []Lock{{Field: "year", Reason: yearLockedMessage}}, Meta: i.Meta,
	}
}
