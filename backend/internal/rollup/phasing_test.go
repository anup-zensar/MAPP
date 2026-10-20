package rollup

import (
	"encoding/json"
	"slices"
	"testing"

	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
)

// amountOf parses an amount or panics.
func amountOf(text string) money.Amount {
	a, err := money.ParseAmount(text)
	if err != nil {
		panic(err)
	}
	return a
}

// budgetOf returns a pointer to the amount text; "" gives nil.
func budgetOf(text string) *money.Amount {
	if text == "" {
		return nil
	}
	a := amountOf(text)
	return &a
}

// texts writes the amounts as digits.
func texts(amounts []money.Amount) []string {
	out := make([]string, 0, len(amounts))
	for _, a := range amounts {
		out = append(out, a.String())
	}
	return out
}

// fields lists the fields of problems, in order.
func fields(problems errs.Problems) []string {
	out := []string{}
	for _, p := range problems {
		out = append(out, p.Field+": "+p.Message)
	}
	return out
}

// Not phased has no monthly amounts (FR-020).
func TestPhasing_None(t *testing.T) {
	if got := NotPhased.MonthAmounts(budgetOf("600000"), 2027); got != nil {
		t.Errorf("none = %v, want no amounts", texts(got))
	}
	if NotPhased.Method != MethodNone || len(NotPhased.Months) != 0 {
		t.Errorf("NotPhased = %+v", NotPhased)
	}
}

// Evenly spreads the budget over the 12 months by the Evenly rule: rounded down, the rest in December; it
// follows every budget change (FR-020, data-model §3.3).
func TestPhasing_Evenly_FR020(t *testing.T) {
	evenly := Phasing{Method: MethodEvenly, Months: []PhasingMonth{}}
	got := evenly.MonthAmounts(budgetOf("600000"), 2027)
	if len(got) != 12 || !slices.ContainsFunc(got, func(a money.Amount) bool { return a.String() == "50000.00" }) {
		t.Fatalf("600,000.00 evenly = %v", texts(got))
	}
	for _, a := range got {
		if a.String() != "50000.00" {
			t.Errorf("600,000.00 evenly = %v, want 50,000.00 each", texts(got))
			break
		}
	}
	got = evenly.MonthAmounts(budgetOf("100000"), 2027)
	if got[0].String() != "8333.33" || got[10].String() != "8333.33" || got[11].String() != "8333.37" {
		t.Errorf("100,000.00 evenly = %v, want 8,333.33 × 11 and 8,333.37", texts(got))
	}
	if got := evenly.MonthAmounts(budgetOf("612000"), 2027); got[2].String() != "51000.00" {
		t.Errorf("612,000.00 evenly: March = %s, want 51,000.00", got[2])
	}
	if got := evenly.MonthAmounts(nil, 2027); got != nil {
		t.Errorf("evenly without a budget = %v, want none", texts(got))
	}
}

// Custom months are the stored months; a month without an entry is 0.00.
func TestPhasing_Custom_MissingMonthsAreZero(t *testing.T) {
	custom := Phasing{Method: MethodCustom, Months: []PhasingMonth{
		{Month: "2027-03", Amount: amountOf("100000")}, {Month: "2027-11", Amount: amountOf("100000")},
	}}
	got := custom.MonthAmounts(budgetOf("400000"), 2027)
	want := []string{"0.00", "0.00", "100000.00", "0.00", "0.00", "0.00", "0.00", "0.00", "0.00", "0.00", "100000.00", "0.00"}
	if !slices.Equal(texts(got), want) {
		t.Errorf("custom = %v, want %v", texts(got), want)
	}
}

// A phasing save is checked: a method, a budget to phase, months of the plan's year only, each once, and
// amounts of 0.00 or more with at most two decimals (FR-020, data-model §3.3).
func TestCheckPhasing_Rules_FR020(t *testing.T) {
	cases := []struct {
		name   string
		in     PhasingInput
		budget string
		want   []string
	}{
		{"no budget", PhasingInput{Method: "evenly"}, "", []string{"method: " + msgSetBudgetFirst}},
		{"no budget, not phased is fine", PhasingInput{Method: "none"}, "", []string{}},
		{"unknown method", PhasingInput{Method: "weekly"}, "100", []string{"method: " + msgChooseMethod}},
		{"month outside the year", PhasingInput{Method: "custom", Months: []PhasingMonthInput{{Month: "2028-01", Amount: "10"}}}, "100",
			[]string{"months: Months must be in 2027."}},
		{"not a month", PhasingInput{Method: "custom", Months: []PhasingMonthInput{{Month: "2027-13", Amount: "10"}}}, "100",
			[]string{"months: Months must be in 2027."}},
		{"negative", PhasingInput{Method: "custom", Months: []PhasingMonthInput{{Month: "2027-01", Amount: "-1"}}}, "100",
			[]string{"months.2027-01: " + msgAmountNegative}},
		{"three decimals", PhasingInput{Method: "custom", Months: []PhasingMonthInput{{Month: "2027-02", Amount: "1.005"}}}, "100",
			[]string{"months.2027-02: " + msgTwoDecimals}},
		{"not an amount", PhasingInput{Method: "custom", Months: []PhasingMonthInput{{Month: "2027-02", Amount: "abc"}}}, "100",
			[]string{"months.2027-02: " + msgEnterAmount}},
		{"duplicate month", PhasingInput{Method: "custom", Months: []PhasingMonthInput{{Month: "2027-02", Amount: "1"}, {Month: "2027-02", Amount: "2"}}}, "100",
			[]string{"months: " + msgEachMonthOnce}},
	}
	for _, c := range cases {
		_, problems := ParsePhasing(c.in, budgetOf(c.budget), 2027)
		if got := fields(problems); !slices.Equal(got, c.want) {
			t.Errorf("%s: problems = %q, want %q", c.name, got, c.want)
		}
	}

	p, problems := ParsePhasing(PhasingInput{Method: "custom", Months: []PhasingMonthInput{
		{Month: "2027-11", Amount: "100000"}, {Month: "2027-03", Amount: " 100000.5 "},
	}}, budgetOf("400000"), 2027)
	if len(problems) > 0 || p.Method != MethodCustom || len(p.Months) != 2 || p.Months[0].Month != "2027-03" ||
		p.Months[0].Amount.String() != "100000.50" {
		t.Errorf("custom = %+v, %v; want the months sorted and parsed", p, problems)
	}
	p, _ = ParsePhasing(PhasingInput{Method: "evenly", Months: []PhasingMonthInput{{Month: "2027-03", Amount: "1"}}}, budgetOf("1"), 2027)
	if len(p.Months) != 0 {
		t.Errorf("evenly keeps months %v, want none", p.Months)
	}
}

// A custom phasing that does not add up to the budget is a needed-to-publish problem of its record; one
// that adds up, or any other method, is none (FR-022).
func TestPhasingProblem_Mismatch_FR022(t *testing.T) {
	custom := Phasing{Method: MethodCustom, Months: []PhasingMonth{{Month: "2027-01", Amount: amountOf("590000")}}}
	problem := PhasingProblem(custom, budgetOf("600000"), 2027, "GBP")
	if problem == nil || problem.Field != "budgetPhasing" || problem.Label != "Budget by month" ||
		problem.Reason != "Adds up to 590,000.00 GBP, not the budget 600,000.00 GBP" {
		t.Errorf("problem = %+v", problem)
	}
	if PhasingProblem(custom, budgetOf("590000"), 2027, "GBP") != nil {
		t.Error("a custom phasing that adds up is a problem")
	}
	if PhasingProblem(Phasing{Method: MethodEvenly}, budgetOf("600000"), 2027, "GBP") != nil ||
		PhasingProblem(NotPhased, budgetOf("600000"), 2027, "GBP") != nil {
		t.Error("evenly or not phased is a problem")
	}
	view := ViewOf(custom, budgetOf("600000"), 2027, "GBP")
	if view.Problem == nil || *view.Problem != problem.Reason || view.Total == nil || view.Total.Amount.String() != "590000.00" ||
		len(view.Monthly) != 12 || view.Monthly[0].Currency != "GBP" || len(view.Months) != 1 {
		t.Errorf("view = %+v", view)
	}
	if none := ViewOf(NotPhased, budgetOf("600000"), 2027, "GBP"); none.Total != nil || none.Problem != nil ||
		len(none.Monthly) != 0 || none.Months == nil {
		t.Errorf("not phased view = %+v", none)
	}
}

// A next-year copy moves every month one year later (FR-025).
func TestShiftPhasing_NextYear_FR025(t *testing.T) {
	custom := Phasing{Method: MethodCustom, Months: []PhasingMonth{{Month: "2027-03", Amount: amountOf("100000")}}}
	shifted := ShiftPhasing(custom, 1)
	if shifted.Months[0].Month != "2028-03" || custom.Months[0].Month != "2027-03" || shifted.Months[0].Amount != amountOf("100000") {
		t.Errorf("shifted = %+v (source %+v)", shifted, custom)
	}
	if got := ShiftPhasing(Phasing{Method: MethodEvenly}, 1); got.Method != MethodEvenly || got.Months == nil {
		t.Errorf("evenly shifted = %+v", got)
	}
}

// A phasing change is logged month by month with old and new amounts in the plan currency (FR-023).
func TestPhasingChanges_MonthByMonth_FR023(t *testing.T) {
	before := Phasing{Method: MethodEvenly}
	after := Phasing{Method: MethodCustom, Months: []PhasingMonth{}}
	for m := 1; m <= 12; m++ {
		amount := "50000"
		if m == 3 {
			amount = "60000"
		}
		after.Months = append(after.Months, PhasingMonth{Month: monthKey(2027, m), Amount: amountOf(amount)})
	}
	changes := PhasingChanges(before, after, budgetOf("600000"), budgetOf("600000"), 2027, "GBP")
	if len(changes) != 2 || changes[0].Field != "budgetPhasing.method" || changes[1].Field != "budgetPhasing.2027-03" {
		t.Fatalf("changes = %+v", changes)
	}
	label, text, ok := DescribePhasing(changes[1].Field, decoded(t, changes[1].Old))
	_, newText, _ := DescribePhasing(changes[1].Field, decoded(t, changes[1].New))
	if !ok || label != "Budget by month: Mar 2027" || text != "50,000.00 GBP" || newText != "60,000.00 GBP" {
		t.Errorf("describe = %q, %q → %q (%v)", label, text, newText, ok)
	}
	label, text, _ = DescribePhasing(changes[0].Field, decoded(t, changes[0].New))
	if label != "Budget by month" || text != "Custom" {
		t.Errorf("method = %q %q", label, text)
	}
	if _, _, ok := DescribePhasing("budget", nil); ok {
		t.Error("DescribePhasing describes another field")
	}
	if got := PhasingChanges(after, after, budgetOf("600000"), budgetOf("600000"), 2027, "GBP"); len(got) != 0 {
		t.Errorf("no change = %+v", got)
	}
}

// decoded decodes a stored value as history does.
func decoded(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	return v
}
