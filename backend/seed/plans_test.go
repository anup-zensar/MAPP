package seed_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"northstar/internal/plans"
	"northstar/internal/settings"
)

// The three demo plans of 004 plan "Seed", P-1001 … P-1003: one published plan of Lubricants, a draft of
// Mobility without a business unit (Publish shows what it needs), and a 2027 AOP plan of both business
// units with a revenue goal (US2).
func TestLoad_Plans_US2(t *testing.T) {
	d := loadDemo(t)
	smp := d.attributeID(t, settings.LevelPlan, "Strategic Marketing Plan (SMP)")
	bu := d.attributeID(t, settings.LevelPlan, "Business unit")

	var got []string
	for _, p := range d.plans {
		var units []string
		for _, id := range optionIDs(t, d.attributeByID[bu], p.Attributes[bu]) {
			units = append(units, d.label(id))
		}
		budget := "none"
		if p.Budget != nil {
			budget = p.Budget.Format()
		}
		got = append(got, fmt.Sprintf("%s %s %d %s %s %s %s %s %s %s:%s fb=%s aop=%v smp=%s spread=%s",
			p.ID, p.Name, p.Year, p.ModelID, strings.Join(p.TeamIDs, ","), p.OwnerID, budget, p.Status,
			strings.Join(units, "+"), p.GoalStage, p.GoalAmount, p.DefaultFinanceBudgetID, p.AOP, raw(p.Attributes[smp]), p.DefaultSpread))
	}
	want := []string{
		"P-1001 FY2026 Lubricants Europe 2026 MOD-1001 T-1001 U-1002 1,200,000.00 published Lubricants inq:250000 fb=FB-1001 aop=false smp= spread=evenly",
		"P-1002 FY2026 Mobility China 2026 MOD-1002 T-1002 U-1004 800,000.00 draft  inq:150000 fb=FB-1002 aop=false smp= spread=evenly",
		"P-1003 FY2027 Global Brand 2027 MOD-1001 T-1003 U-1001 500,000.00 draft Mobility+Lubricants revenue:2000000.00 fb= aop=true smp=true spread=evenly",
		"P-1004 FY2027 Lubricants UK & Ireland 2027 MOD-1001 T-1001 U-1002 600,000.00 draft Lubricants inq: fb= aop=false smp= spread=evenly",
	}
	if !slices.Equal(got, want) {
		t.Errorf("plans =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if d.plans[0].Description != "Lubricants marketing in Europe for 2026." {
		t.Errorf("P-1001 description = %q", d.plans[0].Description)
	}
	for _, p := range d.plans {
		if p.Attributes == nil || p.TeamIDs == nil || p.Status == "" || p.Meta.Version != 1 {
			t.Errorf("plan %s is not complete: %+v", p.ID, p)
		}
		if p.Status != plans.StatusDraft && p.Status != plans.StatusPublished {
			t.Errorf("plan %s has status %q", p.ID, p.Status)
		}
	}
}

// raw writes a stored attribute value as its JSON, "" when there is none.
func raw(value json.RawMessage) string {
	return string(value)
}

// 016's plan type and plan currency (data-model §8): P-1001 and P-1002 Operational, P-1003 Strategic, all USD and
// not phased; P-1004 Cross-market, planned in GBP, its 600,000.00 GBP budget phased evenly.
func TestLoad_PlanTypesAndCurrencies_016(t *testing.T) {
	d := loadDemo(t)
	var got []string
	for _, p := range d.plans {
		got = append(got, fmt.Sprintf("%s %s %s %s", p.ID, p.PlanType, p.Currency, p.BudgetPhasing.Method))
	}
	want := []string{"P-1001 operational USD none", "P-1002 operational USD none", "P-1003 strategic USD none", "P-1004 crossMarket GBP evenly"}
	if !slices.Equal(got, want) {
		t.Errorf("plans = %q, want %q", got, want)
	}
}
