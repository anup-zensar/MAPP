package plans

import (
	"strings"
	"testing"
	"time"

	"northstar/internal/history"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/settings"
)

// User story 5 — Publish and unpublish.

// statusTexts returns the field, old text and new text of an entry's changes.
func statusTexts(e history.Entry) [][3]string {
	var texts [][3]string
	for _, c := range e.Changes {
		texts = append(texts, [3]string{c.Field, c.OldText, c.NewText})
	}
	return texts
}

// A complete Draft plan becomes Published, with a new version and one "statusChanged" entry "published"
// that shows the status going from Draft to Published (FR-024, FR-031, US5 scenario 1).
func TestPublish_CompleteDraft_FR024(t *testing.T) {
	f := newFixture(t)
	plan := f.create(t, priya, input("FY2026 Lubricants Test"))
	f.clock.Advance(time.Hour)

	view, err := f.service.Publish(t.Context(), daniel, plan.ID)

	if err != nil || view.Status != StatusPublished || view.Version != 2 || view.UpdatedBy != daniel {
		t.Fatalf("Publish = %s, version %d, by %+v, %v; want Published, version 2, by Daniel",
			view.Status, view.Version, view.UpdatedBy, err)
	}
	if stored := f.stored(t, plan.ID); stored.Status != StatusPublished || !stored.UpdatedAt.Equal(start.Add(time.Hour)) {
		t.Errorf("stored: %s at %v; want Published an hour later", stored.Status, stored.UpdatedAt)
	}
	entries := f.entries(t)
	if len(entries) != 2 || entries[0].Action != history.ActionStatusChanged || entries[0].Verb != "published" ||
		entries[0].Actor != daniel || entries[0].RecordID != plan.ID {
		t.Fatalf("log = %+v, want a published entry by Daniel after the created one", entries)
	}
	if got := statusTexts(entries[0]); len(got) != 1 || got[0] != [3]string{"status", "Draft", "Published"} {
		t.Errorf("changes = %v, want status Draft → Published", got)
	}
}

// A plan that misses fields is not published: the answer lists every missing field, the plan's own first,
// then each campaign or initiative that blocks publishing, and nothing changes (FR-024, US5 scenario 2).
func TestPublish_ListsMissingGrouped_FR024(t *testing.T) {
	f := newFixture(t)
	in := input("FY2026 Mobility Test")
	in.Attributes = settings.AttributeValues{} // no business unit
	plan := f.create(t, priya, in)
	retail := errs.RecordRef{Type: "campaign", ID: "C-1001", Name: "Retail growth"}
	cluster := settings.MissingField{Field: "attributes.SA-CL", Label: "Cluster"}
	region := settings.MissingField{Field: "attributes.SA-RE", Label: "Region"}
	f.addLevel(t, "campaign", map[string]Facts{plan.ID: {Campaigns: 1, NeedsAttention: []AttentionItem{
		{Record: retail, Missing: []settings.MissingField{cluster, region}, BlocksPublish: true},
	}}})

	_, err := f.service.Publish(t.Context(), priya, plan.ID)

	assertError(t, err, errs.NotReady("Publish needs 3 fields", []errs.ReadinessGroup{
		{Record: errs.RecordRef{Type: "plan", ID: plan.ID, Name: "FY2026 Mobility Test"},
			Fields: []settings.MissingField{{Field: "attributes." + businessUnit, Label: "Business unit"}}},
		{Record: retail, Fields: []settings.MissingField{cluster, region}},
	}))
	if stored := f.stored(t, plan.ID); stored.Status != StatusDraft || stored.Version != 1 {
		t.Errorf("stored: %s, version %d; want Draft, version 1", stored.Status, stored.Version)
	}
	if entries := f.entries(t); len(entries) != 1 {
		t.Errorf("log has %d entries, want only the created one", len(entries))
	}
}

// A plan missing only one field says so in the singular (plan "User-visible messages").
func TestPublish_OneMissingField_FR024(t *testing.T) {
	f := newFixture(t)
	in := input("FY2026 Mobility Test")
	in.Attributes = settings.AttributeValues{}
	plan := f.create(t, priya, in)

	_, err := f.service.Publish(t.Context(), priya, plan.ID)

	assertError(t, err, errs.NotReady("Publish needs 1 field", []errs.ReadinessGroup{
		{Record: errs.RecordRef{Type: "plan", ID: plan.ID, Name: "FY2026 Mobility Test"},
			Fields: []settings.MissingField{{Field: "attributes." + businessUnit, Label: "Business unit"}}},
	}))
}

// Warnings never block publishing: a negative remaining budget, and activities or line items that still
// need fields (FR-024, US5 scenario 4).
func TestPublish_WarningsDoNotBlock_FR024(t *testing.T) {
	f := newFixture(t)
	in := input("FY2026 Lubricants Test")
	in.Budget = &money.Input{Amount: "100000", Currency: "USD"}
	plan := f.create(t, priya, in)
	// One fake level reports the facts of every level below (a fake level replaces a plan's facts).
	f.addLevel(t, "campaign", map[string]Facts{plan.ID: {Campaigns: 1, CampaignBudgets: usd(t, "150000.00"), Activities: 1, NeedsAttention: []AttentionItem{{
		Record:  errs.RecordRef{Type: "activity", ID: "A-1001", Name: "Spring launch"},
		Missing: []settings.MissingField{{Field: "attributes.SA-BRD", Label: "Brands"}},
	}}}})

	view, err := f.service.Publish(t.Context(), priya, plan.ID)

	if err != nil || view.Status != StatusPublished {
		t.Fatalf("Publish = %s, %v; want Published", view.Status, err)
	}
	if view.Figures.OverAllocatedBy == nil || len(view.Completeness.ItemsBelow) != 1 {
		t.Errorf("figures %+v, items below %+v; want over-allocated and the activity still listed", view.Figures, view.Completeness.ItemsBelow)
	}
}

// Publishing a Published plan changes nothing and logs nothing (D-083).
func TestPublish_AlreadyPublished_NoChange(t *testing.T) {
	f := newFixture(t)
	plan := f.create(t, priya, input("FY2026 Lubricants Test"))
	if _, err := f.service.Publish(t.Context(), priya, plan.ID); err != nil {
		t.Fatalf("first Publish: %v", err)
	}

	view, err := f.service.Publish(t.Context(), daniel, plan.ID)

	if err != nil || view.Status != StatusPublished || view.Version != 2 {
		t.Errorf("second Publish = %s, version %d, %v; want Published, version 2", view.Status, view.Version, err)
	}
	if entries := f.entries(t); len(entries) != 2 {
		t.Errorf("log has %d entries, want created and published only", len(entries))
	}
}

// A plan that does not exist can be neither published nor unpublished.
func TestPublishAndUnpublish_NotFound_FR024_FR025(t *testing.T) {
	f := newFixture(t)

	_, publishErr := f.service.Publish(t.Context(), priya, "P-9999")
	_, unpublishErr := f.service.Unpublish(t.Context(), priya, "P-9999", UnpublishRequest{Reason: "Budget review"})

	assertError(t, publishErr, errs.NotFound(planGoneMessage))
	assertError(t, unpublishErr, errs.NotFound(planGoneMessage))
}

// Unpublishing needs a reason of 1 to 500 characters, without its surrounding spaces; a refused unpublish
// changes nothing (FR-025, US5 scenario 3).
func TestUnpublish_ReasonRules_FR025(t *testing.T) {
	f := newFixture(t)
	f.put(t, seedPlan(t, "P-1001", "FY2026 Lubricants Europe", 2026, teamLubricants, priya.ID, "", StatusPublished))
	cases := []struct {
		reason string
		want   *errs.Error
	}{
		{"", problems("reason", "Enter a reason.")},
		{"   ", problems("reason", "Enter a reason.")},
		{strings.Repeat("é", 501), problems("reason", "Use 500 characters or fewer.")},
	}
	for _, c := range cases {
		_, err := f.service.Unpublish(t.Context(), priya, "P-1001", UnpublishRequest{Reason: c.reason})

		assertError(t, err, c.want)
	}
	if stored := f.stored(t, "P-1001"); stored.Status != StatusPublished || stored.Version != 1 {
		t.Errorf("stored: %s, version %d; want Published, version 1", stored.Status, stored.Version)
	}

	view, err := f.service.Unpublish(t.Context(), priya, "P-1001", UnpublishRequest{Reason: " " + strings.Repeat("é", 500) + " "})

	if err != nil || view.Status != StatusDraft {
		t.Errorf("Unpublish with 500 characters = %s, %v; want Draft", view.Status, err)
	}
}

// Unpublishing returns the plan to Draft and records the reason in one "statusChanged" entry
// "unpublished"; unpublishing a Draft plan changes nothing and logs nothing (FR-025, FR-031, D-083).
func TestUnpublish_ReturnsToDraftAndLogsReason_FR025(t *testing.T) {
	f := newFixture(t)
	f.put(t, seedPlan(t, "P-1001", "FY2026 Lubricants Europe", 2026, teamLubricants, priya.ID, "", StatusPublished))
	f.clock.Advance(time.Hour)

	view, err := f.service.Unpublish(t.Context(), daniel, "P-1001", UnpublishRequest{Reason: "  Budget under review  "})
	again, againErr := f.service.Unpublish(t.Context(), daniel, "P-1001", UnpublishRequest{Reason: "Twice"})

	if err != nil || view.Status != StatusDraft || view.Version != 2 || view.UpdatedBy != daniel {
		t.Fatalf("Unpublish = %s, version %d, by %+v, %v; want Draft, version 2, by Daniel", view.Status, view.Version, view.UpdatedBy, err)
	}
	if againErr != nil || again.Status != StatusDraft || again.Version != 2 {
		t.Errorf("second Unpublish = %s, version %d, %v; want Draft, version 2, unchanged", again.Status, again.Version, againErr)
	}
	entries := f.entries(t)
	if len(entries) != 1 || entries[0].Action != history.ActionStatusChanged || entries[0].Verb != "unpublished" ||
		entries[0].Comment != "Budget under review" || entries[0].Actor != daniel {
		t.Fatalf("log = %+v, want one unpublished entry by Daniel with the reason", entries)
	}
	if got := statusTexts(entries[0]); len(got) != 1 || got[0] != [3]string{"status", "Published", "Draft"} {
		t.Errorf("changes = %v, want status Published → Draft", got)
	}
}
