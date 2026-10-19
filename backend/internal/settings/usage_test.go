package settings

import (
	"context"
	"reflect"
	"testing"

	"northstar/internal/history"
)

// "Used by" counts every record using a Settings record, those in the recycle bin included and said so,
// joins the parts with " · ", and lists the live plans of models and finance budgets by name; an unused
// record reads "Not used yet" (FR-003, data model §10).
func TestUsage_SummaryAndRecycleBin_FR003(t *testing.T) {
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	webinar := f.addType(t, "Webinar", true)
	lubricants := f.addModel(t, "Lubricants 1.0", event.ID)
	f.addModel(t, "Mobility 1.0", event.ID)
	tf := f.addAttributes(t,
		testAttribute{level: LevelPlan, label: "Business unit", typ: TypeMultipleChoice, options: []string{"Mobility", "Lubricants"}},
		testAttribute{level: LevelPlan, label: "SMP", typ: TypeYesNo, condition: []string{"Mobility"}},
		testAttribute{level: LevelCampaign, label: "Cluster", typ: TypeMultipleChoice, options: []string{"Europe", "Asia"}},
		testAttribute{level: LevelCampaign, label: "Region", typ: TypeMultipleChoice, parent: "Cluster",
			options: []string{"UK", "DACH", "SEA"}, links: map[string][]string{"UK": {"Europe"}, "DACH": {"Europe"}, "SEA": {"Asia"}}},
		testAttribute{level: LevelActivity, label: "Brands", typ: TypeMultipleChoiceSplit, condition: []string{"Lubricants"}, options: []string{"Helix"}},
	)
	cluster, europe := tf.attribute(LevelCampaign, "Cluster"), tf.option(LevelCampaign, "Cluster", "Europe")
	item := f.addItem(t, 2026, "2026 Lubricants Budget", "Europe – Digital", "", true)
	budget := item.BudgetID
	f.refs = []Reference{
		planRef("P-1002", "Mobility FY2026", lubricants.ID, false, nil),
		planRef("P-1001", "Lubricants FY2026", lubricants.ID, false, nil),
		planRef("P-1003", "Old plan", lubricants.ID, true, nil),
		activityRef("A-1001", "P-1001", event.ID, false, nil),
		activityRef("A-1002", "P-1001", event.ID, true, nil),
		{Record: history.RecordRef{Type: "campaign", ID: "C-1001"}, Level: LevelCampaign, PlanID: "P-1001",
			Attributes: values(cluster.ID, `["`+europe+`"]`)},
		{Record: history.RecordRef{Type: "campaign", ID: "C-1002"}, Level: LevelCampaign, PlanID: "P-1001", InBin: true,
			Attributes: values(cluster.ID, `["`+tf.option(LevelCampaign, "Cluster", "Asia")+`"]`)},
		lineItemRef("L-1001", "P-1001", "A-1001", "EUR", "100.00", false),
		lineItemRef("L-1002", "P-1001", "A-1001", "EUR", "100.00", true),
		personRef("U-1002", "EUR"),
		{Record: history.RecordRef{Type: "lineItem", ID: "L-1003"}, Level: LevelLineItem, FinanceBudgetItemID: item.ID},
		{Record: history.RecordRef{Type: "plan", ID: "P-1004", Name: "Budget plan"}, Level: LevelPlan, PlanID: "P-1004", FinanceBudgetID: budget},
	}

	var u *usageIndex
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		u, err = f.service.usage(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("usage: %v", err)
	}

	tests := []struct {
		name    string
		usage   Usage
		summary string
	}{
		{"model", u.model(lubricants.ID), "Used by 3 plans (1 in the recycle bin)"},
		{"activity type", u.activityType(event.ID), "Used by 2 activities (1 in the recycle bin) · Allowed in 2 models"},
		{"unused activity type", u.activityType(webinar.ID), "Not used yet"},
		{"attribute with values and a dependent", u.attribute(cluster), "Used by 2 campaigns (1 in the recycle bin) · Region depends on it"},
		{"Business unit", u.attribute(tf.attribute(LevelPlan, "Business unit")), "Condition of 2 attributes"},
		{"option", u.option(tf.options["campaign/Cluster/Europe"], cluster), "Used by 1 campaign · Linked to 2 Region options"},
		{"Business unit option", u.option(tf.options["plan/Business unit/Lubricants"], tf.attribute(LevelPlan, "Business unit")), "Condition of 1 attribute"},
		{"currency", u.currency("EUR"), "Used by 2 line items (1 in the recycle bin) · Display currency of 1 person"},
		{"unused currency", u.currency("GBP"), "Not used yet"},
		{"finance budget item", u.item(item.ID), "Linked to 1 line item"},
		{"finance budget", u.budget(budget), "Default finance budget of 1 plan"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.usage.Summary != tt.summary || tt.usage.InUse != (tt.summary != notUsedYet) {
				t.Errorf("usage = %q (in use %v), want %q", tt.usage.Summary, tt.usage.InUse, tt.summary)
			}
		})
	}
	wantPlans := []history.RecordRef{{Type: "plan", ID: "P-1001", Name: "Lubricants FY2026"}, {Type: "plan", ID: "P-1002", Name: "Mobility FY2026"}}
	if records := u.model(lubricants.ID).Records; !reflect.DeepEqual(records, wantPlans) {
		t.Errorf("model records = %+v, want the live plans by name %+v", records, wantPlans)
	}
	wantPart := UsagePart{Kind: "plan", Text: "Used by 3 plans (1 in the recycle bin)", Count: 3, InRecycleBin: 1}
	if parts := u.model(lubricants.ID).Parts; !reflect.DeepEqual(parts, []UsagePart{wantPart}) {
		t.Errorf("model parts = %+v, want %+v", parts, wantPart)
	}
}

// A Settings record in the recycle bin keeps what it uses in use, so a restore never meets a missing
// setting (spec edge case): a model in the bin keeps its activity types, a dependent attribute in the bin
// keeps its parent, and its options keep their parent options.
func TestUsage_SettingsInBinCount_EdgeCase(t *testing.T) {
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	model := f.addModel(t, "Lubricants 1.0", event.ID)
	tf := f.addAttributes(t,
		testAttribute{level: LevelCampaign, label: "Cluster", typ: TypeMultipleChoice, options: []string{"Europe"}},
		testAttribute{level: LevelCampaign, label: "Region", typ: TypeMultipleChoice, parent: "Cluster",
			options: []string{"UK"}, links: map[string][]string{"UK": {"Europe"}}},
	)
	f.write(t, func(ctx context.Context, _ Repository) error {
		if _, err := f.history.MoveToBin(ctx, alex, modelKind, model.ID); err != nil {
			return err
		}
		_, err := f.history.MoveToBin(ctx, alex, attributeKind, tf.attribute(LevelCampaign, "Region").ID)
		return err
	})

	var u *usageIndex
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		u, err = f.service.usage(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("usage: %v", err)
	}

	checks := map[string][2]string{
		"activity type": {u.activityType(event.ID).Summary, "Allowed in 1 model (1 in the recycle bin)"},
		"parent":        {u.attribute(tf.attribute(LevelCampaign, "Cluster")).Summary, "Region depends on it (1 in the recycle bin)"},
		"parent option": {u.option(tf.options["campaign/Cluster/Europe"], tf.attribute(LevelCampaign, "Cluster")).Summary,
			"Linked to 1 Region option (1 in the recycle bin)"},
	}
	for name, c := range checks {
		if c[0] != c[1] {
			t.Errorf("%s: usage = %q, want %q", name, c[0], c[1])
		}
	}
}
