package bulk_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/bulk"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
)

func TestRun_RequestChecks_012FR006_FR007(t *testing.T) {
	t.Parallel()
	many := make([]string, 1001)
	for i := range many {
		many[i] = "A-" + strings.Repeat("1", i+1)
	}
	tests := []struct {
		name string
		req  bulk.Request
		want errs.FieldError
	}{
		{name: "nothing selected", req: bulk.Request{Action: bulk.ActionDelete},
			want: errs.FieldError{Field: "recordIds", Message: "Select at least one record."}},
		{name: "FR-006 over 1,000 records", req: bulk.Request{Action: bulk.ActionDelete, RecordIDs: many},
			want: errs.FieldError{Field: "recordIds", Message: "Bulk actions handle up to 1,000 records at a time — narrow the filters"}},
		{name: "a plan is not selectable", req: bulk.Request{Action: bulk.ActionDelete, RecordIDs: []string{"P-1001"}},
			want: errs.FieldError{Field: "recordIds", Message: "Only campaigns, initiatives and activities can be selected."}},
		{name: "an unknown action", req: bulk.Request{Action: "archive", RecordIDs: []string{"A-1001"}},
			want: errs.FieldError{Field: "action", Message: "Choose one of the listed actions."}},
		{name: "no owner", req: bulk.Request{Action: bulk.ActionChangeOwner, RecordIDs: []string{"A-1001"}},
			want: errs.FieldError{Field: "ownerId", Message: "Choose a person."}},
		{name: "an inactive owner", req: bulk.Request{Action: bulk.ActionChangeOwner, RecordIDs: []string{"A-1001"}, OwnerID: "U-1009"},
			want: errs.FieldError{Field: "ownerId", Message: "Person U-1009 is inactive — choose an active person."}},
		{name: "FR-007 change field on two levels", req: bulk.Request{Action: bulk.ActionChangeField, RecordIDs: []string{"A-1001", "C-1001"},
			Field: &bulk.FieldChange{Key: "startDate"}},
			want: errs.FieldError{Field: "recordIds", Message: "Change field works on one level at a time — keep only one level."}},
		{name: "FR-007 copy on two levels", req: bulk.Request{Action: bulk.ActionCopy, RecordIDs: []string{"I-1001", "C-1001"}, TargetID: "P-1"},
			want: errs.FieldError{Field: "recordIds", Message: "Copy works on one level at a time — keep only one level."}},
		{name: "no target", req: bulk.Request{Action: bulk.ActionMove, RecordIDs: []string{"A-1001"}},
			want: errs.FieldError{Field: "targetId", Message: "Choose where to move them."}},
		{name: "a long comment", req: bulk.Request{Action: bulk.ActionSubmit, RecordIDs: []string{"A-1001"}, Comment: strings.Repeat("x", 2001)},
			want: errs.FieldError{Field: "comment", Message: "Use 2,000 characters or fewer."}},
		{name: "FR-009 a field not offered", req: bulk.Request{Action: bulk.ActionChangeField, RecordIDs: []string{"A-1001"},
			Field: &bulk.FieldChange{Key: "plannedCost"}},
			want: errs.FieldError{Field: "field.key", Message: "plannedCost can't be changed for activities here."}},
		{name: "FR-009 a required field is not cleared", req: bulk.Request{Action: bulk.ActionChangeField, RecordIDs: []string{"A-1001"},
			Field: &bulk.FieldChange{Key: "activityType"}},
			want: errs.FieldError{Field: "field.value", Message: "Activity type is required — it can't be cleared."}},
		{name: "an option of another field", req: bulk.Request{Action: bulk.ActionChangeField, RecordIDs: []string{"A-1001"},
			Field: &bulk.FieldChange{Key: "activityType", Value: &bulk.FieldValue{OptionIDs: []string{"SO-1"}}}},
			want: errs.FieldError{Field: "field.value", Message: "Choose one of the listed options."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			f.planning.fields = []bulk.Field{{Key: "activityType", Label: "Activity type", Kind: "singleChoice", Required: true,
				Options: []bulk.FieldOption{{ID: "AT-1", Label: "Event"}}}}

			_, err := f.service.Run(t.Context(), priya, tt.req)

			assertValidation(t, err, tt.want)
		})
	}
}

func TestChangeOwner_ChangedOrAlreadySet_012FR008_FR021(t *testing.T) {
	t.Parallel()
	owned := activity("A-1002", "Trade fair", bulk.StatusCreated)
	owned.OwnerID = "U-1003"
	f := newFixture(t, container("C-1001", "Helix launch", bulk.LevelCampaign), activity("A-1001", "Launch film", bulk.StatusCreated), owned)

	result, err := f.service.Run(t.Context(), priya, bulk.Request{Action: bulk.ActionChangeOwner, OwnerID: "U-1003",
		RecordIDs: []string{"C-1001", "A-1001", "A-1002", "A-1001"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	id := "BLK-1001"
	gotHead := []any{result.ID, result.Verb, result.Note}
	if diff := cmp.Diff([]any{&id, "changed", "Bulk change of 1 campaign and 2 activities (BLK-1001)"}, gotHead); diff != "" {
		t.Errorf("id, verb and note (-want +got):\n%s", diff)
	}
	want := []bulk.Outcome{
		{RecordID: "C-1001", Level: bulk.LevelCampaign, Name: "Helix launch", PlanID: "P-1001", Outcome: "done", Reasons: []string{}},
		{RecordID: "A-1001", Level: bulk.LevelActivity, Name: "Launch film", PlanID: "P-1001", Outcome: "done", Reasons: []string{}},
		{RecordID: "A-1002", Level: bulk.LevelActivity, Name: "Trade fair", PlanID: "P-1001", Outcome: "skipped",
			Reasons: []string{"Already set — nothing to change"}},
	}
	if diff := cmp.Diff(want, result.Outcomes); diff != "" {
		t.Errorf("outcomes (-want +got):\n%s", diff)
	}
	if result.Counts != (bulk.Counts{Done: 2, Skipped: 1}) || f.stored(t, "A-1001").OwnerID != "U-1003" {
		t.Errorf("counts %+v, owner %s", result.Counts, f.stored(t, "A-1001").OwnerID)
	}
	if _, found, _ := f.store.GetAction(t.Context(), "BLK-1001", priya.ID); !found {
		t.Error("the action is not stored")
	}
}

func TestRun_EveryLogEntryCarriesTheNote_012FR023(t *testing.T) {
	t.Parallel()
	f := newFixture(t, activity("A-1001", "Launch film", bulk.StatusCreated), activity("A-1002", "Trade fair", bulk.StatusCreated))

	if _, err := f.service.Run(t.Context(), priya, bulk.Request{Action: bulk.ActionChangeOwner, OwnerID: "U-1003",
		RecordIDs: []string{"A-1001", "A-1002"}}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	entries := f.log(t)
	if len(entries) != 2 {
		t.Fatalf("%d entries, want one per record", len(entries))
	}
	for _, e := range entries {
		if e.Details != "Bulk change of 2 activities (BLK-1001)" || e.Actor != priya {
			t.Errorf("entry of %s: %q by %v", e.RecordID, e.Details, e.Actor)
		}
	}
}

func TestPreview_SavesNothing_012D126(t *testing.T) {
	t.Parallel()
	f := newFixture(t, activity("A-1001", "Launch film", bulk.StatusCreated))
	req := bulk.Request{Action: bulk.ActionChangeOwner, OwnerID: "U-1003", RecordIDs: []string{"A-1001"}}

	preview, err := f.service.Preview(t.Context(), priya, req)
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}

	gotHead := []any{preview.Preview, preview.ID, preview.Note, preview.Counts}
	if diff := cmp.Diff([]any{true, (*string)(nil), "Bulk change of 1 activity", bulk.Counts{Done: 1}}, gotHead); diff != "" {
		t.Errorf("preview (-want +got):\n%s", diff)
	}
	if f.stored(t, "A-1001").OwnerID != "" || len(f.log(t)) != 0 {
		t.Error("the preview changed the record or wrote the log")
	}
	run, err := f.service.Run(t.Context(), priya, req)
	if err != nil || *run.ID != "BLK-1001" {
		t.Errorf("the run after the preview = %v, %v; want BLK-1001", run.ID, err)
	}
}

func TestRun_GoneFailedOthersKept_UnexpectedErrorSavesNothing_012FR022(t *testing.T) {
	t.Parallel()
	f := newFixture(t, activity("A-1001", "Launch film", bulk.StatusCreated))
	req := bulk.Request{Action: bulk.ActionChangeOwner, OwnerID: "U-1003", RecordIDs: []string{"A-1999", "A-1001"}}

	result, err := f.service.Run(t.Context(), priya, req)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Counts != (bulk.Counts{Done: 1, Failed: 1}) || result.Outcomes[0].Reasons[0] != "No longer exists" {
		t.Errorf("result = %+v", result)
	}

	f.planning.brokenNext = true
	if _, err := f.service.Run(t.Context(), priya, bulk.Request{Action: bulk.ActionChangeOwner, OwnerID: "U-1004",
		RecordIDs: []string{"A-1001"}}); err == nil {
		t.Fatal("a storage failure must fail the run")
	}
	if _, found, _ := f.store.GetAction(t.Context(), "BLK-1002", priya.ID); found {
		t.Error("the failed run was stored")
	}
}

func TestChangeField_ReapprovalAndRefusals_012FR010_FR011(t *testing.T) {
	t.Parallel()
	day := date.New(2026, 6, 30)
	change := &bulk.FieldChange{Key: "endDate", Value: &bulk.FieldValue{Date: &day}}
	tests := []struct {
		name         string
		restart      bool
		wantOutcomes []string
		wantReasons  [][]string
	}{
		{name: "leave those unchanged", wantOutcomes: []string{"done", "skipped", "skipped", "skipped"}, wantReasons: [][]string{{},
			{"Approved — left unchanged so it keeps its approval"}, {"Submitted — left unchanged so it stays with its approvers"},
			{"Dates must be inside the initiative's dates."}}},
		{name: "change and restart approval", restart: true, wantOutcomes: []string{"done", "done", "done", "skipped"},
			wantReasons: [][]string{{}, {}, {}, {"Dates must be inside the initiative's dates."}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t, activity("A-1001", "Launch film", bulk.StatusCreated), activity("A-1002", "Fleet card", bulk.StatusApproved),
				activity("A-1003", "Trade fair", bulk.StatusSubmitted), activity("A-1004", "Podcast", bulk.StatusCreated))
			f.planning.fields = []bulk.Field{{Key: "endDate", Label: "End date", Kind: "date"}}
			f.planning.refusals["A-1004"] = errs.Validation(errs.FieldError{Field: "endDate", Message: "Dates must be inside the initiative's dates."})

			result, err := f.service.Run(t.Context(), priya, bulk.Request{Action: bulk.ActionChangeField, Field: change,
				RecordIDs: []string{"A-1001", "A-1002", "A-1003", "A-1004"}, RestartApproval: tt.restart})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}

			var outcomes []string
			var reasons [][]string
			for _, o := range result.Outcomes {
				outcomes, reasons = append(outcomes, o.Outcome), append(reasons, o.Reasons)
			}
			if diff := cmp.Diff(tt.wantOutcomes, outcomes); diff != "" {
				t.Errorf("outcomes (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantReasons, reasons); diff != "" {
				t.Errorf("reasons (-want +got):\n%s", diff)
			}
			if len(result.Reapproval) != 2 || result.Reapproval[0].RecordID != "A-1002" || result.Reapproval[1].RecordID != "A-1003" {
				t.Errorf("reapproval = %+v", result.Reapproval)
			}
		})
	}
}

func TestSubmit_ActivitiesOnlyEveryReasonAtOnce_012FR012_FR013(t *testing.T) {
	t.Parallel()
	f := newFixture(t, container("C-1001", "Helix launch", bulk.LevelCampaign), activity("A-1001", "Launch film", bulk.StatusCreated),
		activity("A-1002", "Fleet card", bulk.StatusApproved), activity("A-1003", "Trade fair", bulk.StatusSubmitted),
		activity("A-1004", "Podcast", bulk.StatusCreated))
	f.planning.refusals["A-1004"] = errs.NotReady("Fill in the missing fields first.", []errs.ReadinessGroup{
		{Record: errs.RecordRef{Type: "activity", ID: "A-1004"}, Fields: []errs.MissingField{{Label: "Owner"}, {Label: "Company code"}}},
		{Record: errs.RecordRef{Type: "line item", ID: "L-1004"}, Fields: []errs.MissingField{{Label: "GL code"}}},
	})

	result, err := f.service.Run(t.Context(), priya, bulk.Request{Action: bulk.ActionSubmit,
		RecordIDs: []string{"C-1001", "A-1001", "A-1002", "A-1003", "A-1004"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	got := map[string][]string{}
	for _, o := range result.Outcomes {
		got[o.RecordID] = append([]string{o.Outcome}, o.Reasons...)
	}
	want := map[string][]string{
		"C-1001": {"skipped", "Only activities go through approval"}, "A-1001": {"done"},
		"A-1002": {"skipped", "Already approved"}, "A-1003": {"skipped", "Already submitted"},
		"A-1004": {"skipped", "2 fields needed: Owner, Company code; line item L-1004: GL code"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("outcomes (-want +got):\n%s", diff)
	}
	if result.Verb != "submitted" || len(f.planning.submitted) != 1 {
		t.Errorf("verb %s, submitted %v", result.Verb, f.planning.submitted)
	}
}

func TestDelete_NestedCountedOnceAndSummarised_012FR019_FR020(t *testing.T) {
	t.Parallel()
	elsewhere := activity("A-1005", "Trade fair", bulk.StatusApproved)
	elsewhere.Ancestors = []string{"I-1009", "C-1001"}
	f := newFixture(t, container("I-1001", "Events", bulk.LevelInitiative, "C-1001"), activity("A-1001", "Launch film", bulk.StatusCreated),
		elsewhere)
	f.planning.below["I-1001"] = bulk.Below{Counts: bulk.BelowCounts{Activities: 2, LineItems: 5}, InApproval: 1}
	f.planning.below["A-1005"] = bulk.Below{Counts: bulk.BelowCounts{LineItems: 1}, InApproval: 1}

	result, err := f.service.Run(t.Context(), priya, bulk.Request{Action: bulk.ActionDelete, RecordIDs: []string{"I-1001", "A-1001", "A-1005"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	wantSummary := &bulk.DeleteSummary{Selected: bulk.LevelCounts{Initiatives: 1, Activities: 1},
		Below: bulk.BelowCounts{Activities: 2, LineItems: 6}, SubmittedOrApproved: 2}
	if diff := cmp.Diff(wantSummary, result.DeleteSummary); diff != "" {
		t.Errorf("summary (-want +got):\n%s", diff)
	}
	if len(result.RecycleBinEntries) != 2 || result.Counts.Done != 2 || result.UndoAvailableUntil == nil ||
		!result.UndoAvailableUntil.Equal(start.Add(2*time.Minute)) {
		t.Errorf("entries %d, counts %+v, undo until %v", len(result.RecycleBinEntries), result.Counts, result.UndoAvailableUntil)
	}
	action, _, _ := f.store.GetAction(t.Context(), "BLK-1001", priya.ID)
	if diff := cmp.Diff([]string{"BIN-I-1001", "BIN-A-1005"}, action.BinEntryIDs); diff != "" {
		t.Errorf("stored bin entries (-want +got):\n%s", diff)
	}
}

func TestUndo_Refusals_012FR020(t *testing.T) {
	t.Parallel()
	f := newFixture(t, activity("A-1001", "Launch film", bulk.StatusCreated))
	if _, err := f.service.Run(t.Context(), priya, bulk.Request{Action: bulk.ActionChangeOwner, OwnerID: "U-1003",
		RecordIDs: []string{"A-1001"}}); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, id, code string
	}{
		{name: "not a delete", id: "BLK-1001", code: errs.CodeBadRequest},
		{name: "unknown", id: "BLK-1999", code: errs.CodeNotFound},
	}
	for _, tt := range tests {
		_, err := f.service.Undo(t.Context(), priya, tt.id)
		if got, ok := errors.AsType[*errs.Error](err); !ok || got.Code != tt.code {
			t.Errorf("%s: error %v, want %s", tt.name, err, tt.code)
		}
	}
	if _, err := f.service.Undo(t.Context(), actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}, "BLK-1001"); !isCode(err, errs.CodeNotFound) {
		t.Errorf("someone else's action: %v, want not_found", err)
	}
}

func isCode(err error, code string) bool {
	got, ok := errors.AsType[*errs.Error](err)
	return ok && got.Code == code
}
