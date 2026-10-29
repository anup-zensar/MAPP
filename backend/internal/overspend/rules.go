package overspend

import (
	"cmp"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

const (
	CodeOverBudget = "over_budget"
	MsgChooseMode  = "Choose Warn or Refuse."

	msgPhrase   = "the planned cost of %s '%s' would be %s, over its budget of %s by %s"
	msgOneOwner = "Not saved — %s. Its owner refuses overspending."
	msgOwners   = "Not saved — %s. Their owners refuse overspending."
)

// ParseMode reads a mode a request sends; false for anything but "warn" and "refuse".
func ParseMode(text string) (Mode, bool) {
	switch Mode(text) {
	case Warn, Refuse:
		return Mode(text), true
	}
	return Warn, false
}

type budgetKey struct {
	kind Kind
	id   string
}

func keyOf(b Budget) budgetKey {
	return budgetKey{kind: b.Kind, id: b.ID}
}

// Overages lists the budgets set to Refuse that a save took over (FR-009, FR-011, research R2): over after the
// save, and either new (no budget before) or with more planned cost or a smaller amount than before.
func Overages(before, after []Budget) []Overage {
	known := make(map[budgetKey]Budget, len(before))
	for _, b := range before {
		known[keyOf(b)] = b
	}
	overages := []Overage{}
	for _, b := range after {
		if !b.Mode.Refuses() || !isOver(b) {
			continue
		}
		if was, found := known[keyOf(b)]; found && !isWorse(was, b) {
			continue
		}
		overages = append(overages, overageOf(b))
	}
	slices.SortFunc(overages, byLevelAndName)
	return overages
}

func isOver(b Budget) bool {
	return b.Planned.Amount.Cmp(b.Amount.Amount) > 0
}

func isWorse(was, now Budget) bool {
	return now.Planned.Amount.Cmp(was.Planned.Amount) > 0 || now.Amount.Amount.Cmp(was.Amount.Amount) < 0
}

func overageOf(b Budget) Overage {
	over := money.Money{Amount: b.Planned.Amount.Sub(b.Amount.Amount), Currency: b.Amount.Currency}
	return Overage{Budget: refOf(b), Amount: b.Amount, PlannedCost: b.Planned, OverBy: over}
}

func refOf(b Budget) errs.RecordRef {
	return errs.RecordRef{Type: string(b.Kind), ID: b.ID, Name: b.Name}
}

func byLevelAndName(x, y Overage) int {
	return cmp.Or(cmp.Compare(kindOrder[Kind(x.Budget.Type)], kindOrder[Kind(y.Budget.Type)]),
		strings.Compare(x.Budget.Name, y.Budget.Name), strings.Compare(x.Budget.ID, y.Budget.ID))
}

// Refusal is the answer of a refused save, 409 over_budget, naming every budget it would take over (FR-010).
func Refusal(overages []Overage) *errs.Error {
	phrases := make([]string, 0, len(overages))
	for _, o := range overages {
		phrases = append(phrases, fmt.Sprintf(msgPhrase, kindWords[Kind(o.Budget.Type)], o.Budget.Name,
			o.PlannedCost.Format(), o.Amount.Format(), o.OverBy.Format()))
	}
	format := msgOneOwner
	if len(phrases) > 1 {
		format = msgOwners
	}
	message := fmt.Sprintf(format, strings.Join(phrases, "; "))
	return &errs.Error{Status: http.StatusConflict, Code: CodeOverBudget, Message: message, Fields: []errs.FieldError{}}
}

// LimitsOf returns the budgets set to Refuse on path with what is left to plan in each, in the plan currency
// (FR-014); a global budget's is converted from USD at rates.
func LimitsOf(budgets []Budget, path Path, rates money.Rates) ([]Limit, error) {
	limits := []Limit{}
	for _, b := range budgets {
		if !b.Mode.Refuses() || !isOnPath(b, path) {
			continue
		}
		left := money.Money{Amount: b.Amount.Amount.Sub(b.Planned.Amount), Currency: b.Amount.Currency}
		if left.Currency != path.PlanCurrency {
			converted, err := rates.Convert(left, path.PlanCurrency)
			if err != nil {
				return nil, err
			}
			left = converted
		}
		limits = append(limits, Limit{Budget: refOf(b), LeftToPlan: left})
	}
	slices.SortFunc(limits, func(x, y Limit) int {
		return byLevelAndName(Overage{Budget: x.Budget}, Overage{Budget: y.Budget})
	})
	return limits, nil
}

func isOnPath(b Budget, path Path) bool {
	switch b.Kind {
	case KindPlan:
		return b.ID == path.PlanID
	case KindCampaign:
		return b.ID == path.CampaignID
	case KindInitiative:
		return b.ID == path.InitiativeID
	case KindGlobalBudget:
		return slices.Contains(b.PlanIDs, path.PlanID)
	}
	return false
}
