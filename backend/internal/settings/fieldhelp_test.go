package settings

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
)

// The catalogue has every key once, a label, a known level and a default of at most 300 characters (data-model
// §6, FR-045).
func TestCatalogue_KeysUniqueAndTextsWithinLimit(t *testing.T) {
	seen := map[string]bool{}
	levels := map[HelpLevel]bool{HelpPlan: true, HelpCampaign: true, HelpInitiative: true, HelpActivity: true, HelpLineItem: true,
		HelpPhasing: true, HelpCommon: true, HelpAllWork: true}
	for _, d := range helpCatalogue {
		if seen[d.Key] {
			t.Errorf("%s is in the catalogue twice", d.Key)
		}
		seen[d.Key] = true
		if d.Label == "" || d.Default == "" || !levels[d.Level] || !strings.HasPrefix(d.Key, string(d.Level)+".") {
			t.Errorf("entry %+v", d)
		}
		if n := utf8.RuneCountInString(d.Default); n > maxHelpText {
			t.Errorf("%s: default of %d characters", d.Key, n)
		}
	}
	for _, key := range []string{"plan.budget", "plan.currency", "plan.planType", "activity.plannedCost", "activity.splitFrom",
		"lineItem.plannedCostPlan", "phasing.allocated", "common.id", "allWork.breakdownShare"} {
		if !seen[key] {
			t.Errorf("the catalogue has no %s", key)
		}
	}
}

// Field help lists every catalogue entry in order with its default text, version 0 and no editor, or its
// edited text with who edited it (FR-045, US7-1).
func TestListFieldHelp_DefaultsAndEdited_FR045(t *testing.T) {
	f := newTestService(t)
	if _, err := f.service.UpdateFieldHelp(t.Context(), alex, "activity.plannedCost", FieldHelpUpdate{Text: text("Edited."), Version: version(0)}); err != nil {
		t.Fatalf("UpdateFieldHelp: %v", err)
	}

	page, err := f.service.ListFieldHelp(t.Context(), daniel)

	if err != nil || page.Total != len(helpCatalogue) || len(page.Items) != len(helpCatalogue) {
		t.Fatalf("ListFieldHelp = %d entries, %v", len(page.Items), err)
	}
	first := page.Items[0]
	if first.Key != "plan.name" || first.Level != HelpPlan || first.Label != "Plan name" || !first.IsDefault || first.Version != 0 ||
		first.Text != first.DefaultText || first.UpdatedAt != nil || first.UpdatedBy != nil {
		t.Errorf("first entry = %+v", first)
	}
	budget := entryOf(t, page, "plan.budget")
	if budget.Text != "The money this plan may spend in its year, in the plan currency. Campaign budgets are allocated from it." {
		t.Errorf("plan.budget = %q", budget.Text)
	}
	edited := entryOf(t, page, "activity.plannedCost")
	if edited.Text != "Edited." || edited.IsDefault || edited.Version != 1 || *edited.UpdatedBy != alex || !edited.UpdatedAt.Equal(start) {
		t.Errorf("edited entry = %+v", edited)
	}
}

// entryOf returns the entry of key on page.
func entryOf(t *testing.T, page FieldHelpPage, key string) FieldHelpEntry {
	t.Helper()
	for _, e := range page.Items {
		if e.Key == key {
			return e
		}
	}
	t.Fatalf("no entry %s", key)
	return FieldHelpEntry{}
}

// An edit is saved as plain text — trimmed, line breaks made spaces — and logged with the old and new text;
// an empty text is allowed and hides the ⓘ (FR-046, US7-2, US7-4).
func TestUpdateFieldHelp_SavesAndLogs_FR046(t *testing.T) {
	f := newTestService(t)

	saved, err := f.service.UpdateFieldHelp(t.Context(), alex, "plan.budget", FieldHelpUpdate{Text: text(" The budget\nof the plan. "), Version: version(0)})
	entry := f.lastEntry(t)
	f.clock.now = start.Add(time.Hour)
	emptied, emptyErr := f.service.UpdateFieldHelp(t.Context(), daniel, "plan.budget", FieldHelpUpdate{Text: text(""), Version: version(1)})

	if err != nil || saved.Text != "The budget of the plan." || saved.IsDefault || saved.Version != 1 || *saved.UpdatedBy != alex {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
	want := []history.Change{{Field: "text", Label: "Help text",
		Old:     json.RawMessage(`"The money this plan may spend in its year, in the plan currency. Campaign budgets are allocated from it."`),
		New:     json.RawMessage(`"The budget of the plan."`),
		OldText: "The money this plan may spend in its year, in the plan currency. Campaign budgets are allocated from it.",
		NewText: "The budget of the plan."}}
	if entry.RecordType != "fieldHelp" || entry.RecordID != "plan.budget" || entry.RecordName != "Plan › Plan budget" ||
		entry.Action != history.ActionChanged || entry.Actor != alex || !reflect.DeepEqual(entry.Changes, want) {
		t.Errorf("log entry = %+v", entry)
	}
	if emptyErr != nil || emptied.Text != "" || emptied.Version != 2 || *emptied.UpdatedBy != daniel {
		t.Errorf("emptied = %+v, %v", emptied, emptyErr)
	}
}

// Reset to default deletes the edited text — the default is back, version 0 — and logs the change (FR-046,
// US7-3). Resetting a default changes nothing.
func TestUpdateFieldHelp_ResetDeletesRow_FR046(t *testing.T) {
	f := newTestService(t)
	if _, err := f.service.UpdateFieldHelp(t.Context(), alex, "plan.budget", FieldHelpUpdate{Text: text("Edited."), Version: version(0)}); err != nil {
		t.Fatalf("UpdateFieldHelp: %v", err)
	}

	reset, err := f.service.UpdateFieldHelp(t.Context(), daniel, "plan.budget", FieldHelpUpdate{Text: nil, Version: version(1)})
	entries := len(f.logEntries(t))
	again, againErr := f.service.UpdateFieldHelp(t.Context(), daniel, "plan.budget", FieldHelpUpdate{Text: nil, Version: version(0)})

	if err != nil || !reset.IsDefault || reset.Version != 0 || reset.Text != reset.DefaultText || reset.UpdatedBy != nil {
		t.Fatalf("reset = %+v, %v", reset, err)
	}
	last := f.lastEntry(t)
	if last.Actor != daniel || len(last.Changes) != 1 || last.Changes[0].OldText != "Edited." || last.Changes[0].NewText != reset.DefaultText {
		t.Errorf("log entry = %+v", last)
	}
	if againErr != nil || !again.IsDefault || len(f.logEntries(t)) != entries {
		t.Errorf("resetting a default = %+v, %v (log %d → %d)", again, againErr, entries, len(f.logEntries(t)))
	}
}

// A text above 300 characters is refused (FR-046, US7-4).
func TestUpdateFieldHelp_TooLong_FR046(t *testing.T) {
	f := newTestService(t)

	_, err := f.service.UpdateFieldHelp(t.Context(), alex, "plan.budget", FieldHelpUpdate{Text: text(strings.Repeat("a", 301)), Version: version(0)})
	_, noVersion := f.service.UpdateFieldHelp(t.Context(), alex, "plan.budget", FieldHelpUpdate{Text: text("ok")})

	assertError(t, err, problems("text", "Use 300 characters or fewer."))
	assertError(t, noVersion, problems("version", "Reload the page and try again."))
}

// Version 0 is the default: a save from it after someone edited the text is refused naming who and when; so
// is a save from an old version, and from an edited version after a reset (FR-046, 001 FR-029).
func TestUpdateFieldHelp_Version0Conflict_FR046(t *testing.T) {
	f := newTestService(t)
	f.clock.now = start.Add(time.Hour)
	if _, err := f.service.UpdateFieldHelp(t.Context(), daniel, "plan.budget", FieldHelpUpdate{Text: text("Daniel's."), Version: version(0)}); err != nil {
		t.Fatalf("UpdateFieldHelp: %v", err)
	}

	_, fromDefault := f.service.UpdateFieldHelp(t.Context(), alex, "plan.budget", FieldHelpUpdate{Text: text("Alex's."), Version: version(0)})
	_, fromOld := f.service.UpdateFieldHelp(t.Context(), alex, "plan.budget", FieldHelpUpdate{Text: text("Alex's."), Version: version(2)})
	f.clock.now = start.Add(2 * time.Hour)
	if _, err := f.service.UpdateFieldHelp(t.Context(), daniel, "plan.budget", FieldHelpUpdate{Version: version(1)}); err != nil {
		t.Fatalf("reset: %v", err)
	}
	_, afterReset := f.service.UpdateFieldHelp(t.Context(), alex, "plan.budget", FieldHelpUpdate{Text: text("Alex's."), Version: version(1)})

	assertError(t, fromDefault, errs.VersionConflict("field help", "Daniel Okoro", start.Add(time.Hour)))
	assertError(t, fromOld, errs.VersionConflict("field help", "Daniel Okoro", start.Add(time.Hour)))
	assertError(t, afterReset, errs.VersionConflict("field help", "Daniel Okoro", start.Add(2*time.Hour)))
}

// A key outside the catalogue answers not found (data-model §2.5).
func TestUpdateFieldHelp_UnknownKey404(t *testing.T) {
	f := newTestService(t)

	_, err := f.service.UpdateFieldHelp(t.Context(), alex, "plan.colour", FieldHelpUpdate{Text: text("x"), Version: version(0)})

	var apiErr *errs.Error
	if !errors.As(err, &apiErr) || apiErr.Code != errs.CodeNotFound || apiErr.Message != "This help entry doesn't exist." {
		t.Errorf("UpdateFieldHelp(plan.colour) = %v", err)
	}
}

// Field help counts the active Shell attributes without help, for its note and link (FR-048, US7-5).
func TestListFieldHelp_AttributesWithoutHelp_FR048(t *testing.T) {
	f := newTestService(t)
	f.addAttributes(t,
		testAttribute{level: LevelPlan, label: "Business unit", typ: TypeMultipleChoice, options: []string{"Mobility"}, help: "The business."},
		testAttribute{level: LevelCampaign, label: "Brand", typ: TypeSingleChoice, options: []string{"Helix"}},
		testAttribute{level: LevelActivity, label: "Channel", typ: TypeSingleChoice, options: []string{"TV"}, help: "  "},
		testAttribute{level: LevelActivity, label: "Old", typ: TypeSingleChoice, options: []string{"x"}, inactive: true},
	)

	page, err := f.service.ListFieldHelp(t.Context(), alex)

	if err != nil || page.ShellAttributesWithoutHelp != 2 {
		t.Errorf("ShellAttributesWithoutHelp = %d, %v; want 2 (Brand, Channel)", page.ShellAttributesWithoutHelp, err)
	}
}
