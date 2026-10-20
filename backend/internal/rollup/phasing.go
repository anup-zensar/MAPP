package rollup

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"northstar/internal/history"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
)

// Budget phasing (016 FR-020 – FR-025, data-model §3.3, D-265): a plan's, campaign's or initiative's budget
// split over the 12 months of the plan's year. Only the method and, for Custom, the typed months are stored;
// the monthly amounts are worked out on read. Every amount is in the plan currency.

// PhasingMethod is how a budget is phased (contract PhasingMethod).
type PhasingMethod string

// The phasing methods.
const (
	MethodNone   PhasingMethod = "none"   // not phased (the default)
	MethodEvenly PhasingMethod = "evenly" // the budget over 12 months by the Evenly rule, following the budget
	MethodCustom PhasingMethod = "custom" // an amount per month, as typed
)

// PhasingMonth is one typed month of a Custom phasing: "2027-03" and its amount in the plan currency.
type PhasingMonth struct {
	Month  string       `json:"month"`
	Amount money.Amount `json:"amount"`
}

// Phasing is a budget's phasing as stored on the record (data-model §4.1, §4.2). Months is empty unless
// the method is Custom; sorted, one entry per month.
type Phasing struct {
	Method PhasingMethod  `json:"method"`
	Months []PhasingMonth `json:"months"`
}

// NotPhased is the phasing of a record that has none: the default, and what removing a budget leaves.
var NotPhased = Phasing{Method: MethodNone, Months: []PhasingMonth{}}

// Normalised returns p with the defaults of older data (data-model §4.5): no method reads as Not phased,
// and Months is never nil.
func (p Phasing) Normalised() Phasing {
	if p.Method == "" {
		p.Method = MethodNone
	}
	if p.Months == nil || p.Method != MethodCustom {
		p.Months = []PhasingMonth{}
	}
	return p
}

// MonthAmounts returns the 12 monthly amounts, January to December of year: none (nil) when not phased or
// without a budget; the Evenly rule of domain model §4.4 (rounded down, the rest in December) for Evenly;
// the typed months, 0.00 for a month not typed, for Custom.
func (p Phasing) MonthAmounts(budget *money.Amount, year int) []money.Amount {
	switch p.Method {
	case MethodEvenly:
		if budget == nil {
			return nil
		}
		return budget.SplitEvenly(12)
	case MethodCustom:
		months := make([]money.Amount, 12)
		for _, m := range p.Months {
			if y, month, ok := parseMonth(m.Month); ok && y == year {
				months[month-1] = months[month-1].Add(m.Amount)
			}
		}
		return months
	}
	return nil
}

// The messages of a phasing save (016 plan "User-visible messages").
const (
	msgChooseMethod   = "Choose Not phased, Evenly or Custom."
	msgSetBudgetFirst = "Set a budget before phasing it."
	msgMonthsInYear   = "Months must be in %d."
	msgEachMonthOnce  = "Enter each month once."
	msgEnterAmount    = "Enter an amount, for example 50000 or 50,000.00."
	msgTwoDecimals    = "Use at most 2 decimals."
	msgAmountNegative = "An amount can't be negative."
	msgPhasingAddsUp  = "Adds up to %s, not the budget %s"
)

// PhasingInput is the body of the three PUT …/budget-phasing operations (contract BudgetPhasingUpdate):
// amounts as decimal text in the plan currency, so every problem becomes a field message.
type PhasingInput struct {
	Method  string              `json:"method"`
	Months  []PhasingMonthInput `json:"months"`
	Version *int                `json:"version"`
}

// PhasingMonthInput is one month of a PhasingInput (contract PhasingMonthInput).
type PhasingMonthInput struct {
	Month  string `json:"month"`
	Amount string `json:"amount"`
}

// ParsePhasing checks a phasing save for a record with budget in a plan of year (FR-020, data-model §3.3)
// and returns the phasing to store with every problem at once: a known method; a budget for any method but
// Not phased ("Set a budget before phasing it."); for Custom, months of the plan's year only, each once, with
// amounts of 0.00 or more and at most two decimals (fields "months" and "months.<month>"). Months sent with
// another method are ignored.
func ParsePhasing(in PhasingInput, budget *money.Amount, year int) (Phasing, errs.Problems) {
	var problems errs.Problems
	method := PhasingMethod(strings.TrimSpace(in.Method))
	switch method {
	case MethodNone, MethodEvenly, MethodCustom:
		if method != MethodNone && budget == nil {
			problems.Add("method", msgSetBudgetFirst)
		}
	default:
		problems.Add("method", msgChooseMethod)
	}
	p := Phasing{Method: method, Months: []PhasingMonth{}}
	if method != MethodCustom {
		return p, problems
	}
	seen := map[string]bool{}
	outside, twice := false, false
	for _, m := range in.Months {
		key := strings.TrimSpace(m.Month)
		y, _, ok := parseMonth(key)
		if !ok || y != year {
			outside = true
			continue
		}
		if seen[key] {
			twice = true
			continue
		}
		seen[key] = true
		amount, err := money.ParseAmount(strings.TrimSpace(m.Amount))
		switch {
		case errors.Is(err, money.ErrTooManyDecimals):
			problems.Add("months."+key, msgTwoDecimals)
		case err != nil:
			problems.Add("months."+key, msgEnterAmount)
		case amount.Sign() < 0:
			problems.Add("months."+key, msgAmountNegative)
		default:
			p.Months = append(p.Months, PhasingMonth{Month: key, Amount: amount})
		}
	}
	if outside {
		problems.Add("months", fmt.Sprintf(msgMonthsInYear, year))
	}
	if twice {
		problems.Add("months", msgEachMonthOnce)
	}
	slices.SortFunc(p.Months, func(a, b PhasingMonth) int { return cmp.Compare(a.Month, b.Month) })
	return p, problems
}

// PhasingProblem is the needed-to-publish problem of a Custom phasing whose months do not add up to the
// budget (FR-022): "Adds up to 590,000.00 GBP, not the budget 600,000.00 GBP". Nil for every other phasing.
// It blocks publishing the plan, never saving.
func PhasingProblem(p Phasing, budget *money.Amount, year int, currency string) *errs.MissingField {
	if p.Method != MethodCustom || budget == nil {
		return nil
	}
	total := money.Sum(p.MonthAmounts(budget, year)...)
	if total == *budget {
		return nil
	}
	return &errs.MissingField{
		Field: "budgetPhasing", Label: "Budget by month",
		Reason: fmt.Sprintf(msgPhasingAddsUp, money.Money{Amount: total, Currency: currency}.Format(),
			money.Money{Amount: *budget, Currency: currency}.Format()),
	}
}

// ShiftPhasing returns p moved years later, for a copy into another year (FR-025); p is not changed.
func ShiftPhasing(p Phasing, years int) Phasing {
	p = p.Normalised()
	shifted := Phasing{Method: p.Method, Months: make([]PhasingMonth, 0, len(p.Months))}
	for _, m := range p.Months {
		if y, month, ok := parseMonth(m.Month); ok {
			m.Month = monthKey(y+years, month)
		}
		shifted.Months = append(shifted.Months, m)
	}
	return shifted
}

// PhasingMonthView is one typed month as the API answers it (contract PhasingMonth).
type PhasingMonthView struct {
	Month  string      `json:"month"`
	Amount money.Money `json:"amount"`
}

// PhasingView is a phasing as the API answers it (contract BudgetPhasing), in the plan currency.
type PhasingView struct {
	Method  PhasingMethod      `json:"method"`
	Months  []PhasingMonthView `json:"months"`  // Custom only: the stored months
	Monthly []money.Money      `json:"monthly"` // the 12 monthly amounts; empty when not phased
	Total   *money.Money       `json:"total"`   // the sum of Monthly; null when not phased
	Problem *string            `json:"problem"` // the FR-022 problem; null when fine
}

// ViewOf returns the phasing p of a record with budget in a plan of year, in the plan currency.
func ViewOf(p Phasing, budget *money.Amount, year int, currency string) PhasingView {
	p = p.Normalised()
	view := PhasingView{Method: p.Method, Months: []PhasingMonthView{}, Monthly: []money.Money{}}
	for _, m := range p.Months {
		view.Months = append(view.Months, PhasingMonthView{Month: m.Month, Amount: money.Money{Amount: m.Amount, Currency: currency}})
	}
	if amounts := p.MonthAmounts(budget, year); amounts != nil {
		for _, a := range amounts {
			view.Monthly = append(view.Monthly, money.Money{Amount: a, Currency: currency})
		}
		total := money.Money{Amount: money.Sum(amounts...), Currency: currency}
		view.Total = &total
	}
	if problem := PhasingProblem(p, budget, year, currency); problem != nil {
		view.Problem = &problem.Reason
	}
	return view
}

// The activity-log fields of a phasing change (data-model §7): "budgetPhasing.method" and one
// "budgetPhasing.<month>" per changed month, whose values are Money objects in the plan currency.
const (
	phasingField       = "budgetPhasing"
	phasingMethodField = phasingField + ".method"
	phasingLabel       = "Budget by month"
)

// methodWords are the methods as people read them.
var methodWords = map[string]string{
	string(MethodNone): "Not phased", string(MethodEvenly): "Evenly", string(MethodCustom): "Custom",
}

// PhasingChanges lists what changed between two phasings of a record, for its "changed" log entry
// (FR-023): the method, then every month whose amount changed, January first, with old and new amounts in
// the plan currency ("Budget by month: Mar 2027 50,000.00 → 60,000.00 GBP"). Empty when nothing changed.
func PhasingChanges(before, after Phasing, budgetBefore, budgetAfter *money.Amount, year int, currency string) []history.Change {
	before, after = before.Normalised(), after.Normalised()
	changes := []history.Change{}
	if before.Method != after.Method {
		changes = append(changes, history.Change{Field: phasingMethodField, Old: jsonOf(before.Method), New: jsonOf(after.Method)})
	}
	old, now := before.MonthAmounts(budgetBefore, year), after.MonthAmounts(budgetAfter, year)
	for m := 1; m <= 12; m++ {
		oldValue, newValue := monthValue(old, m, currency), monthValue(now, m, currency)
		if string(oldValue) != string(newValue) {
			changes = append(changes, history.Change{Field: phasingField + "." + monthKey(year, m), Old: oldValue, New: newValue})
		}
	}
	return changes
}

// monthValue is the stored value of month m (1–12) of amounts: a Money object, or null when not phased.
func monthValue(amounts []money.Amount, m int, currency string) json.RawMessage {
	if amounts == nil {
		return json.RawMessage("null")
	}
	return jsonOf(money.Money{Amount: amounts[m-1], Currency: currency})
}

// DescribePhasing gives the label and text of a phasing field of the activity log; ok is false for other
// fields. The kinds of plans, campaigns and initiatives call it from their Describe.
func DescribePhasing(field string, value any) (label, text string, ok bool) {
	if !strings.HasPrefix(field, phasingField+".") {
		return "", "", false
	}
	if field == phasingMethodField {
		method, _ := value.(string)
		return phasingLabel, methodWords[method], true
	}
	label = phasingLabel
	if y, month, valid := parseMonth(strings.TrimPrefix(field, phasingField+".")); valid {
		label = fmt.Sprintf("%s: %s %d", phasingLabel, time.Month(month).String()[:3], y)
	}
	return label, MoneyText(value), true
}

// MoneyText writes a stored Money object ({"amount": "50000.00", "currency": "GBP"}) as people read it,
// "50,000.00 GBP"; "" when value is not one (the activity log then uses its default).
func MoneyText(value any) string {
	object, _ := value.(map[string]any)
	text, _ := object["amount"].(string)
	currency, _ := object["currency"].(string)
	amount, err := money.ParseAmount(text)
	if err != nil || currency == "" {
		return ""
	}
	return money.Money{Amount: amount, Currency: currency}.Format()
}

// jsonOf encodes v, which is always encodable here.
func jsonOf(v any) json.RawMessage {
	encoded, _ := json.Marshal(v)
	return encoded
}

// parseMonth reads a month key "2027-03": its year and month (1–12), and false when it is not one.
func parseMonth(key string) (year, month int, ok bool) {
	y, m, found := strings.Cut(key, "-")
	if !found || len(y) != 4 || len(m) != 2 {
		return 0, 0, false
	}
	year, errY := strconv.Atoi(y)
	month, errM := strconv.Atoi(m)
	if errY != nil || errM != nil || month < 1 || month > 12 {
		return 0, 0, false
	}
	return year, month, true
}

// monthKey writes the key of month m (1–12) of year: "2027-03".
func monthKey(year, m int) string {
	return fmt.Sprintf("%04d-%02d", year, m)
}

// MonthKeys returns the 12 month keys of year, "2027-01" … "2027-12".
func MonthKeys(year int) []string {
	keys := make([]string, 0, 12)
	for m := 1; m <= 12; m++ {
		keys = append(keys, monthKey(year, m))
	}
	return keys
}
