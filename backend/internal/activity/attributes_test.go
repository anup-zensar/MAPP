package activity_test

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"mapp/internal/activity"
	"mapp/internal/plan/plantest"
	"mapp/internal/platform/money"
	"mapp/internal/settings/settingstest"
)

// setBusinessUnit sets plan P-1001's Business unit (Mobility SO-1001, Lubricants SO-1002), as a plan edit would.
func (f *fixture) setBusinessUnit(t *testing.T, optionIDs ...string) {
	t.Helper()
	repo := plantest.NewStore(f.db)
	f.change(t, func(ctx context.Context) error {
		p, _, err := repo.Get(ctx, "P-1001")
		if err != nil {
			return err
		}
		encoded, _ := json.Marshal(optionIDs)
		p.Attributes["SA-1001"] = encoded
		return repo.Put(ctx, p)
	})
}

func fieldsOf(v activity.View) []string {
	var fields []string
	for _, p := range v.Readiness.Activity {
		fields = append(fields, p.Field)
	}
	return fields
}

// Hidden business-unit values are kept when a save leaves them out, not needed, and count again once shown.
func TestUpdate_HiddenGroupValuesKept_FR015(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	lubricantsValues := []string{brands, "SA-1023", "SA-1024", "SA-1025", "SA-1026"}
	f.setBusinessUnit(t, "SO-1001")

	view := f.mustEdit(t, "A-1001", func(u *activity.Update) {
		for _, id := range lubricantsValues {
			delete(u.Attributes, id) // the form does not show the Lubricants group
		}
		u.Description = "Saved while the plan is Mobility only."
	})

	for _, id := range lubricantsValues {
		if _, kept := view.Attributes[id]; !kept {
			t.Errorf("the hidden value of %s was lost", id)
		}
	}
	f.mustEdit(t, "A-1001", func(u *activity.Update) { delete(u.Attributes, brands) })
	if fields := fieldsOf(f.get(t, "A-1001")); slices.Contains(fields, "attributes."+brands) {
		t.Errorf("hidden Brands counted as needed: %q", fields)
	}

	f.setBusinessUnit(t, "SO-1002")
	view = f.get(t, "A-1001")
	if _, kept := view.Attributes[brands]; !kept || slices.Contains(fieldsOf(view), "attributes."+brands) {
		t.Errorf("Brands after Lubricants is back: kept %v, problems %q", kept, fieldsOf(view))
	}
	view = f.mustEdit(t, "A-1001", func(u *activity.Update) { delete(u.Attributes, brands) })
	if _, kept := view.Attributes[brands]; kept || !slices.Contains(fieldsOf(view), "attributes."+brands) {
		t.Errorf("a shown value left out is removed and needed again: kept %v, problems %q", kept, fieldsOf(view))
	}
}

// A split not adding up to exactly 100.00 % saves, but blocks submitting; an even split completes it.
func TestUpdate_SplitNot100SavesButIsListed_FR017(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	view := f.mustEdit(t, "A-1001", func(u *activity.Update) {
		u.Attributes[countrySplit] = split("SO-1034", "50.00", "SO-1032", "33.33", "SO-1033", "33.34")
	})

	if !slices.ContainsFunc(view.Readiness.Activity, func(p activity.SubmitProblem) bool {
		return p.Field == "attributes."+countrySplit && p.Message == "Country / OU shares add up to 116.67 % — they must add up to 100.00 %"
	}) {
		t.Errorf("readiness = %+v", view.Readiness.Activity)
	}
	view = f.mustEdit(t, "A-1001", func(u *activity.Update) {
		u.Attributes[countrySplit] = split("SO-1034", "33.33", "SO-1032", "33.33", "SO-1033", "33.34")
	})
	if slices.Contains(fieldsOf(view), "attributes."+countrySplit) {
		t.Errorf("an even split is complete, but the check lists it: %q", fieldsOf(view))
	}
}

// Calculated goals always use the model's current values; typed goals stay as typed.
func TestView_GoalsUseCurrentModelValues_FR018(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	repo := settingstest.NewStore(f.db)
	f.change(t, func(ctx context.Context) error {
		model, _, err := repo.GetModel(ctx, "MOD-1001")
		if err != nil {
			return err
		}
		model.InqToMqlRate = money.PercentFromHundredths(400)
		return repo.PutModel(ctx, model)
	})

	calculated := f.get(t, "A-1001").Goals
	typed := f.get(t, "A-1009").Goals

	if *calculated.MQL != 10_000 || *calculated.Deals != 500 || calculated.Revenue.Format() != "5,000,000.00 USD" || !calculated.Calculated {
		t.Errorf("calculated goals = MQL %d, deals %d, revenue %v", *calculated.MQL, *calculated.Deals, calculated.Revenue)
	}
	if *typed.MQL != 2_000 || typed.Revenue.Format() != "400,000.00 USD" || typed.Deals != nil || typed.Calculated {
		t.Errorf("typed goals = %+v", typed)
	}
}

// Turning "Set later stages myself" off drops the typed values: MQL and revenue are calculated again.
func TestUpdate_TypedGoalsTurnedOff_FR018(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	view := f.mustEdit(t, "A-1009", func(u *activity.Update) { u.Goals.SetLaterStagesMyself = false })

	stored, _ := f.stored(t, "A-1009")
	if stored.MQL != nil || stored.Revenue != nil || !view.Goals.Calculated || *view.Goals.MQL != 1_800 {
		t.Errorf("goals = %+v, stored MQL %v; want calculated 3.00 %% of 60,000 = 1,800", view.Goals, stored.MQL)
	}
}
