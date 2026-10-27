package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// Split activity (feature 016, FR-026 – FR-031, data-model §3.6, D-267): one activity becomes two inside its
// plan. Whole line items move to the new activity and keep their IDs; part of the planned cost moves with
// them; goals stay on the original; the new activity remembers where it came from (splitFromId). Splitting a
// Submitted or Approved activity follows the re-approval rule of domain model §2.5. No undo.

// The messages of a split (016 plan "User-visible messages", data-model §3.6).
const (
	splitSuffix           = " (split)"
	msgSplitNothing       = "Choose line items or an amount to move."
	msgSplitTooMuch       = "Move at most %s, the activity's planned cost."
	msgSplitNoCost        = "This activity has no planned cost."
	msgSplitNotOfActivity = "%s is not a line item of this activity."
	msgSplitLineItemTwice = "Choose each line item once."
	splitVerb             = "split"
	splitIntoDetails      = "Into '%s' (%s): %s"
	splitFromDetails      = "Split from '%s' (%s)"
	splitRestartDetails   = "Approval restarted because the activity was split."
)

// SplitLineItem is a line item the Split dialog lists (ActivitySplitLineItem).
type SplitLineItem struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	PlannedCost     money.Money `json:"plannedCost"`     // in the line item's currency
	PlannedCostPlan money.Money `json:"plannedCostPlan"` // in the plan currency
}

// SplitOptions are what the Split dialog offers (ActivitySplitOptions, FR-026).
type SplitOptions struct {
	ActivityID    string          `json:"activityId"`
	Version       int             `json:"version"`
	Status        Status          `json:"status"`
	PlannedCost   *money.Money    `json:"plannedCost"`
	PlanCurrency  string          `json:"planCurrency"`
	SuggestedName string          `json:"suggestedName"`
	NeedsRestart  bool            `json:"needsRestart"`
	LineItems     []SplitLineItem `json:"lineItems"`
	Destinations  []Destination   `json:"destinations"`
}

// SplitRequest is the body of the split preview and of the split (ActivitySplitRequest).
type SplitRequest struct {
	Name            string   `json:"name"`
	InitiativeID    string   `json:"initiativeId"`
	LineItemIDs     []string `json:"lineItemIds"`
	PlannedCost     *string  `json:"plannedCost"` // in the plan currency; null or "0" moves none
	RestartApproval bool     `json:"restartApproval"`
	Version         *int     `json:"version"`
}

// SplitSide is one activity as the split would leave it (ActivitySplitSide, FR-027).
type SplitSide struct {
	Name                  string       `json:"name"`
	InitiativeName        string       `json:"initiativeName"`
	PlannedCost           *money.Money `json:"plannedCost"`
	LineItemNames         []string     `json:"lineItemNames"`
	UnassignedPlannedCost *money.Money `json:"unassignedPlannedCost"` // null without a planned cost
	UnassignedState       string       `json:"unassignedState"`       // left, allAssigned, exceeded or noPlannedCost
	Status                Status       `json:"status"`
}

// SplitPreview is both activities as the split would leave them, and every problem it would answer
// (ActivitySplitPreview). A side is nil when it cannot be worked out.
type SplitPreview struct {
	Original *SplitSide        `json:"original"`
	Created  *SplitSide        `json:"created"`
	Problems []errs.FieldError `json:"problems"`
}

// SplitResult is the original and the new activity after a split (ActivitySplitResult).
type SplitResult struct {
	Activity    View `json:"activity"`
	NewActivity View `json:"newActivity"`
}

// The states of an unassigned planned cost (as feature 007's balance).
const (
	unassignedLeft          = "left"
	unassignedAllAssigned   = "allAssigned"
	unassignedExceeded      = "exceeded"
	unassignedNoPlannedCost = "noPlannedCost"
)

// SplitOptions returns what the Split dialog of the activity id offers (FR-026): its line items with their
// amounts in the plan currency, the suggested name "<name> (split)" free in its initiative, whether the split
// restarts its approval, and the initiatives of the plan — its own first — with Move's date rule as the
// reason an initiative can't take the new activity.
func (s *Service) SplitOptions(ctx context.Context, a actor.Actor, id string) (SplitOptions, error) {
	destinations, err := s.Destinations(ctx, a, id) // its own Read
	if err != nil {
		return SplitOptions{}, err
	}
	var options SplitOptions
	err = s.tx.Read(ctx, func(ctx context.Context) error {
		activity, pc, err := s.load(ctx, id)
		if err != nil {
			return err
		}
		items, err := s.lineItems.Items(ctx, id)
		if err != nil {
			return err
		}
		siblings, err := s.repo.ListByInitiative(ctx, activity.InitiativeID)
		if err != nil {
			return err
		}
		options = SplitOptions{
			ActivityID: activity.ID, Version: activity.Version, Status: activity.Status, PlannedCost: activity.PlannedCost,
			PlanCurrency: pc.plan.Currency, SuggestedName: suffixedName(siblings, activity.Name, splitSuffix),
			NeedsRestart: inApproval(activity), LineItems: make([]SplitLineItem, 0, len(items)),
			Destinations: splitDestinations(destinations),
		}
		for _, li := range items {
			options.LineItems = append(options.LineItems, SplitLineItem{ID: li.ID, Name: li.Name, PlannedCost: li.PlannedCost, PlannedCostPlan: li.PlannedCostPlan})
		}
		return nil
	})
	if err != nil {
		return SplitOptions{}, err
	}
	return options, nil
}

// splitDestinations turns Move's destinations into the Split dialog's: the activity's own initiative first
// and available; elsewhere only the date rule applies, because the new activity has a name of its own.
func splitDestinations(all []Destination) []Destination {
	list := make([]Destination, 0, len(all))
	for _, d := range all {
		d.MoveProblem = d.CopyProblem // the dates; nil for its own initiative
		if d.Current {
			list = append([]Destination{d}, list...)
			continue
		}
		list = append(list, d)
	}
	return list
}

// suffixedName returns the first of "<name><suffix>", "<name><suffix without its parenthesis> 2)", … that no
// activity among siblings has, shortened to fit 120 characters.
func suffixedName(siblings []Activity, name, suffix string) string {
	for n := 1; ; n++ {
		end := suffix
		if n > 1 {
			end = fmt.Sprintf("%s %d)", strings.TrimSuffix(suffix, ")"), n)
		}
		base := name
		if room := maxNameLength - utf8.RuneCountInString(end); utf8.RuneCountInString(base) > room {
			base = strings.TrimRight(string([]rune(base)[:room]), " ")
		}
		if _, taken := namedLike(siblings, base+end, ""); !taken {
			return base + end
		}
	}
}

// splitPlan is a split worked out: what it changes, and the problems that stop it.
type splitPlan struct {
	original  Activity
	pc        planContext
	target    planContext // the initiative of the new activity
	name      string
	moved     []LineItemBrief // the ticked line items, in the activity's order
	kept      []LineItemBrief
	amount    *money.Money // the planned cost that moves; nil when the original has none
	restart   bool         // the original is Submitted or Approved
	problems  errs.Problems
	sidesKnow bool // the target and the amount are known: the sides can be worked out
}

// planSplit checks a split of the activity id (data-model §3.6) and works out what it changes, inside the
// caller's transaction. Every problem is collected; a missing activity is not found.
func (s *Service) planSplit(ctx context.Context, id string, in SplitRequest) (splitPlan, error) {
	original, pc, err := s.load(ctx, id)
	if err != nil {
		return splitPlan{}, err
	}
	p := splitPlan{original: original, pc: pc, restart: inApproval(original)}

	targetKnown := false
	target, found, err := s.contextOf(ctx, strings.TrimSpace(in.InitiativeID))
	switch {
	case err != nil:
		return splitPlan{}, err
	case !found || target.plan.ID != pc.plan.ID:
		p.problems.Add("initiativeId", chooseAnotherMessage)
	case target.initiative.ID != original.InitiativeID && !datesFit(original, target.initiative):
		p.problems.Add("initiativeId", datesDontFit(original, target.initiative))
	default:
		p.target, targetKnown = target, true
	}

	p.name = strings.TrimSpace(in.Name)
	switch {
	case p.name == "":
		p.problems.Add("name", enterNameMessage)
	case utf8.RuneCountInString(p.name) > maxNameLength:
		p.problems.Add("name", nameTooLongMessage)
	case targetKnown:
		siblings, err := s.repo.ListByInitiative(ctx, target.initiative.ID)
		if err != nil {
			return splitPlan{}, err
		}
		if other, taken := namedLike(siblings, p.name, ""); taken {
			p.problems.Add("name", fmt.Sprintf(nameTakenFormat, target.initiative.Name, other.Name))
		}
	}

	items, err := s.lineItems.Items(ctx, id)
	if err != nil {
		return splitPlan{}, err
	}
	chosen := map[string]bool{}
	for _, liID := range in.LineItemIDs {
		liID = strings.TrimSpace(liID)
		switch {
		case chosen[liID]:
			p.problems.Add("lineItemIds", msgSplitLineItemTwice)
		case !slices.ContainsFunc(items, func(li LineItemBrief) bool { return li.ID == liID }):
			p.problems.Add("lineItemIds", fmt.Sprintf(msgSplitNotOfActivity, liID))
		}
		chosen[liID] = true
	}
	for _, li := range items {
		if chosen[li.ID] {
			p.moved = append(p.moved, li)
		} else {
			p.kept = append(p.kept, li)
		}
	}

	amountKnown := true
	amount, amountProblem := splitAmount(in.PlannedCost, original.PlannedCost, pc.plan.Currency)
	if amountProblem != "" {
		p.problems.Add("plannedCost", amountProblem)
		amountKnown = false
	}
	p.amount = amount
	if len(p.moved) == 0 && (amount == nil || amount.Amount.Sign() == 0) && amountProblem == "" {
		p.problems.Add("lineItemIds", msgSplitNothing)
	}
	if p.restart && !in.RestartApproval {
		p.problems.Add("restartApproval", restartApprovalMessage)
	}
	p.sidesKnow = targetKnown && amountKnown
	return p, nil
}

// splitAmount reads the planned cost to move (data-model §3.6): 0.00 up to the original's planned cost, in the
// plan currency; none (nil, "" or 0) when the original has no planned cost. It returns the amount — nil when
// the original has none — or the problem.
func splitAmount(text *string, original *money.Money, currency string) (*money.Money, string) {
	typed := ""
	if text != nil {
		typed = strings.TrimSpace(*text)
	}
	var amount money.Amount
	if typed != "" {
		parsed, err := money.ParseAmount(typed)
		if err != nil {
			return nil, enterAmountMessage
		}
		amount = parsed
	}
	if original == nil {
		if !amount.IsZero() {
			return nil, msgSplitNoCost
		}
		return nil, ""
	}
	if amount.Sign() < 0 || amount.Cmp(original.Amount) > 0 {
		return nil, fmt.Sprintf(msgSplitTooMuch, original.Format())
	}
	return &money.Money{Amount: amount, Currency: currency}, ""
}

// sides works out both activities as the split would leave them (FR-027): planned costs, line items, the
// unassigned planned cost in the plan currency (red when negative) and the status.
func (p splitPlan) sides() (original, created SplitSide) {
	var remaining *money.Money
	if p.original.PlannedCost != nil {
		left := money.Money{Amount: p.original.PlannedCost.Amount, Currency: p.pc.plan.Currency}
		if p.amount != nil {
			left.Amount = left.Amount.Sub(p.amount.Amount)
		}
		remaining = &left
	}
	status := p.original.Status
	if p.restart {
		status = StatusCreated
	}
	original = side(p.original.Name, p.pc.initiative.Name, remaining, p.kept, status, p.pc.plan.Currency)
	created = side(p.name, p.target.initiative.Name, p.amount, p.moved, StatusCreated, p.pc.plan.Currency)
	return original, created
}

// side is one activity of a split: its unassigned planned cost is the planned cost minus its line items in
// the plan currency.
func side(name, initiative string, planned *money.Money, items []LineItemBrief, status Status, currency string) SplitSide {
	s := SplitSide{Name: name, InitiativeName: initiative, PlannedCost: planned, LineItemNames: []string{}, Status: status,
		UnassignedState: unassignedNoPlannedCost}
	var assigned money.Amount
	for _, li := range items {
		s.LineItemNames = append(s.LineItemNames, li.Name)
		assigned = assigned.Add(li.PlannedCostPlan.Amount)
	}
	if planned == nil {
		return s
	}
	balance := money.Money{Amount: planned.Amount.Sub(assigned), Currency: currency}
	s.UnassignedPlannedCost = &balance
	switch balance.Amount.Sign() {
	case 1:
		s.UnassignedState = unassignedLeft
	case 0:
		s.UnassignedState = unassignedAllAssigned
	default:
		s.UnassignedState = unassignedExceeded
	}
	return s
}

// PreviewSplit returns both activities as the split of in would leave them, with every problem the split
// would answer, and saves nothing (FR-027). A wrong version is not a problem here: the split itself answers it.
func (s *Service) PreviewSplit(ctx context.Context, a actor.Actor, id string, in SplitRequest) (SplitPreview, error) {
	preview := SplitPreview{Problems: []errs.FieldError{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		p, err := s.planSplit(ctx, id, in)
		if err != nil {
			return err
		}
		for _, problem := range p.problems {
			preview.Problems = append(preview.Problems, errs.FieldError{Field: problem.Field, Message: problem.Message})
		}
		if p.sidesKnow {
			original, created := p.sides()
			preview.Original, preview.Created = &original, &created
		}
		return nil
	})
	if err != nil {
		return SplitPreview{}, err
	}
	return preview, nil
}

// Split splits the activity id, all or nothing (FR-028 – FR-030): the new activity — a copy of the original's
// type, owner, dates, ongoing, workflow, description, external name and Shell attributes, with the chosen name,
// the moved planned cost, no goals, status Created and the split link — the ticked line items moved to it with
// their IDs, values and spreads, the original's planned cost lowered by the moved amount, and the original back
// to Created when it was Submitted or Approved (FR-029). Each change is logged: "split" on the original,
// "created … Split from" on the new activity, "moved" on each line item.
func (s *Service) Split(ctx context.Context, a actor.Actor, id string, in SplitRequest) (SplitResult, error) {
	var result SplitResult
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		p, err := s.planSplit(ctx, id, in)
		if err != nil {
			return err
		}
		if in.Version == nil || *in.Version < 1 {
			p.problems.Add("version", reloadMessage)
		}
		if err := p.problems.Err(); err != nil {
			return err
		}
		if err := p.original.CheckVersion(*in.Version, noun); err != nil {
			return err
		}

		created := newSplitActivity(p)
		if created.ID, err = s.ids.Next(ctx, ids.PrefixActivity); err != nil {
			return err
		}
		created.Meta = record.NewMeta(a, s.now())
		if err := s.repo.Put(ctx, created); err != nil {
			return err
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: kindName, RecordID: created.ID, RecordName: created.Name, Action: history.ActionCreated,
			Details: fmt.Sprintf(splitFromDetails, p.original.Name, p.original.ID),
		}); err != nil {
			return err
		}
		movedIDs := make([]string, 0, len(p.moved))
		for _, li := range p.moved {
			movedIDs = append(movedIDs, li.ID)
		}
		if len(movedIDs) > 0 {
			if err := s.lineItems.MoveToActivity(ctx, a, movedIDs, created.ID); err != nil {
				return err
			}
		}

		before := p.original
		after := before
		if p.amount != nil && before.PlannedCost != nil {
			left := money.Money{Amount: before.PlannedCost.Amount.Sub(p.amount.Amount), Currency: before.PlannedCost.Currency}
			after.PlannedCost = &left
		}
		if p.restart {
			after.Status = StatusCreated
		}
		after.Touch(a, s.now())
		if err := s.repo.Put(ctx, after); err != nil {
			return err
		}
		var changes []history.Change
		if !samePlannedCost(before.PlannedCost, after.PlannedCost) {
			changes = append(changes, history.Change{Field: "plannedCost", Old: mustJSON(before.PlannedCost), New: mustJSON(after.PlannedCost)})
		}
		entry, err := s.history.Log(ctx, a, history.LogInput{
			Kind: kindName, RecordID: after.ID, RecordName: after.Name, Action: history.ActionChanged, Verb: splitVerb,
			Changes: changes, Details: fmt.Sprintf(splitIntoDetails, created.Name, created.ID, movedText(s, len(p.moved), p.amount)),
		})
		if err != nil {
			return err
		}
		if p.restart {
			if err = s.logRestart(ctx, a, restartLog{before: before, after: after, changes: entry.Changes, details: splitRestartDetails}); err != nil {
				return err
			}
		}

		if result.Activity, err = s.viewOf(ctx, after, p.pc); err != nil {
			return err
		}
		result.NewActivity, err = s.viewOf(ctx, created, p.target)
		return err
	})
	if err != nil {
		return SplitResult{}, err
	}
	if result.Activity, err = s.shown(ctx, a, result.Activity); err != nil {
		return SplitResult{}, err
	}
	if result.NewActivity, err = s.shown(ctx, a, result.NewActivity); err != nil {
		return SplitResult{}, err
	}
	return result, nil
}

// newSplitActivity is the new activity of a split (FR-028), without its ID and record facts.
func newSplitActivity(p splitPlan) Activity {
	o := p.original
	created := Activity{
		InitiativeID: p.target.initiative.ID, Name: p.name, ExternalName: o.ExternalName, ActivityTypeID: o.ActivityTypeID,
		OwnerID: o.OwnerID, StartDate: clonePtr(o.StartDate), EndDate: clonePtr(o.EndDate), Ongoing: o.Ongoing,
		ApprovalWorkflowID: o.ApprovalWorkflowID, Description: o.Description, Status: StatusCreated, SplitFromID: o.ID,
		Attributes: cloneValues(o.Attributes),
	}
	if p.amount != nil {
		moved := *p.amount
		created.PlannedCost = &moved
	}
	return created
}

// cloneValues returns a copy of Shell attribute values that shares nothing with them; never nil.
func cloneValues(values settings.AttributeValues) settings.AttributeValues {
	clone := settings.AttributeValues{}
	for id, v := range values {
		clone[id] = append(json.RawMessage{}, v...)
	}
	return clone
}

// movedText says what moved: "1 line item and 19,750.00 GBP", "2 line items", "5,000.00 GBP".
func movedText(s *Service, lineItems int, amount *money.Money) string {
	var parts []string
	if with := s.history.CountsText(map[string]int{lineItemKind: lineItems}); with != "" {
		parts = append(parts, strings.TrimPrefix(with, "with ")) // "with 1 line item" → "1 line item"
	}
	if amount != nil && !amount.Amount.IsZero() {
		parts = append(parts, amount.Format())
	}
	return strings.Join(parts, " and ")
}

// samePlannedCost reports whether two planned costs are the same.
func samePlannedCost(x, y *money.Money) bool {
	if x == nil || y == nil {
		return x == nil && y == nil
	}
	return *x == *y
}

// splitLinks returns the split links of the activity a of the plan planID (FR-031): the activity it was split
// from, and the activities split off it — live ones and those in the recycle bin, "deleted".
func (s *Service) splitLinks(ctx context.Context, a Activity, planID string) (*SplitLink, []SplitLink, error) {
	into := []SplitLink{}
	binned, err := s.history.SnapshotsInBin(ctx, kindName)
	if err != nil {
		return nil, nil, err
	}
	var from *SplitLink
	if a.SplitFromID != "" {
		link := SplitLink{ID: a.SplitFromID, Name: a.SplitFromID, PlanID: planID, Deleted: true}
		if other, found, err := s.repo.Get(ctx, a.SplitFromID); err != nil {
			return nil, nil, err
		} else if found {
			link.Name, link.Deleted = other.Name, false
		} else if snapshot, inBin := snapshotOf(binned, a.SplitFromID); inBin {
			link.Name = snapshot.Name
		}
		from = &link
	}
	all, err := s.splitOff(ctx, a.ID)
	if err != nil {
		return nil, nil, err
	}
	for _, other := range all {
		into = append(into, SplitLink{ID: other.ID, Name: other.Name, PlanID: planID})
	}
	for _, snapshot := range binned {
		var stored Activity
		if err := json.Unmarshal(snapshot.Data, &stored); err != nil {
			return nil, nil, err
		}
		if stored.SplitFromID == a.ID {
			into = append(into, SplitLink{ID: stored.ID, Name: stored.Name, PlanID: planID, Deleted: true})
		}
	}
	slices.SortFunc(into, func(x, y SplitLink) int { return compareIDs(x.ID, y.ID) })
	return from, into, nil
}

// splitOff returns the stored activities split off the activity id.
func (s *Service) splitOff(ctx context.Context, id string) ([]Activity, error) {
	return s.repo.ListSplitFrom(ctx, id)
}

// snapshotOf returns the snapshot of the record id among snapshots.
func snapshotOf(snapshots []history.Snapshot, id string) (history.Snapshot, bool) {
	for _, snapshot := range snapshots {
		if snapshot.ID == id {
			return snapshot, true
		}
	}
	return history.Snapshot{}, false
}

// compareIDs orders readable IDs as the store does: shorter first, then alphabetically.
func compareIDs(x, y string) int {
	if len(x) != len(y) {
		return len(x) - len(y)
	}
	return strings.Compare(x, y)
}

// splitFromID is an activity's split link as the plan tree answers it: nil when it was not split from another.
func splitFromID(a Activity) *string {
	if a.SplitFromID == "" {
		return nil
	}
	id := a.SplitFromID
	return &id
}
