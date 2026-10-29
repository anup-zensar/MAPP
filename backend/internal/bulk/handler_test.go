package bulk_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/bulk"
	"mapp/internal/platform/apitest"
)

const (
	priyasEmail = "priya.raman@shell.com"
	sarasEmail  = "sara.novak@shell.com"
	alexsEmail  = "alex.morgan@shell.com"
	runPath     = "/api/v1/bulk-actions"
	previewPath = "/api/v1/bulk-actions/preview"
	danielID    = "U-1003"
)

type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decoded[T any](t *testing.T, response *apitest.Response, status int) T {
	t.Helper()
	if response.Status != status {
		t.Fatalf("status %d %s, want %d", response.Status, response.Body, status)
	}
	var v T
	response.Decode(t, &v)
	return v
}

func outcomesOf(result bulk.Result) map[string]string {
	got := map[string]string{}
	for _, o := range result.Outcomes {
		got[o.RecordID] = o.Outcome + ": " + strings.Join(o.Reasons, " | ")
	}
	return got
}

func TestFields_Contract_012FR009(t *testing.T) {
	t.Parallel()
	priya := apitest.New(t).SignIn(t, priyasEmail)

	list := decoded[bulk.FieldList](t, priya.Do(t, http.MethodGet, "/api/v1/bulk-actions/fields?level=activity", nil), http.StatusOK)

	keys := []string{}
	offered := []string{"date", "singleChoice", "multipleChoice", "yesNo"}
	for _, f := range list.Items {
		keys = append(keys, f.Key)
		if !slices.Contains(offered, f.Kind) {
			t.Errorf("field %s has the kind %s, which Change field does not offer", f.Label, f.Kind)
		}
	}
	if !slices.Contains(keys, "activityType") || len(keys) < 6 || list.Total != len(keys) {
		t.Errorf("fields = %v", keys)
	}
	refused := priya.Do(t, http.MethodGet, "/api/v1/bulk-actions/fields?level=plan", nil, apitest.SkipRequestValidation())
	if body := decoded[apiError](t, refused, http.StatusBadRequest); body.Error.Code != "validation_failed" {
		t.Errorf("level plan = %+v", body)
	}
}

func TestRun_ChangeOwner_Contract_012FR008_FR023(t *testing.T) {
	t.Parallel()
	priya := apitest.New(t).SignIn(t, priyasEmail)
	body := map[string]any{"action": "changeOwner", "ownerId": danielID, "recordIds": []string{"A-1001", "A-1005"}}

	preview := decoded[bulk.Result](t, priya.Do(t, http.MethodPost, previewPath, body), http.StatusOK)
	run := decoded[bulk.Result](t, priya.Do(t, http.MethodPost, runPath, body), http.StatusCreated)

	if !preview.Preview || preview.Counts.Done != 2 || run.ID == nil || run.Counts.Done != 2 {
		t.Fatalf("preview %+v, run %+v", preview.Counts, run.Counts)
	}
	activity := decoded[struct {
		Owner struct{ ID string } `json:"owner"`
	}](t, priya.Do(t, http.MethodGet, "/api/v1/activities/A-1001", nil, apitest.SkipValidation()), http.StatusOK)
	if activity.Owner.ID != danielID {
		t.Errorf("owner = %s, want Daniel Okoro", activity.Owner.ID)
	}
	log := decoded[struct {
		Items []struct{ Details string } `json:"items"`
	}](t, priya.Do(t, http.MethodGet, "/api/v1/activity-log?recordId=A-1001", nil, apitest.SkipValidation()), http.StatusOK)
	if len(log.Items) == 0 || !strings.Contains(log.Items[0].Details, "Bulk change of 2 activities ("+*run.ID+")") {
		t.Errorf("newest entry of A-1001 = %+v", log.Items)
	}
}

func TestRun_ForbiddenSkippedUnseenFailed_Contract_012FR024(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	sara := server.SignIn(t, sarasEmail)

	result := decoded[bulk.Result](t, sara.Do(t, http.MethodPost, runPath,
		map[string]any{"action": "changeOwner", "ownerId": danielID, "recordIds": []string{"A-1001", "A-9999"}}), http.StatusCreated)

	got := outcomesOf(result)
	if !strings.HasPrefix(got["A-1001"], "skipped: ") || got["A-9999"] != "failed: No longer exists" {
		t.Errorf("outcomes = %v; want the viewer skipped with a reason and the unknown failed", got)
	}
}

func TestRun_ChangeField_Contract_012FR010_FR011(t *testing.T) {
	t.Parallel()
	priya := apitest.New(t).SignIn(t, priyasEmail)
	body := map[string]any{"action": "changeField", "recordIds": []string{"A-1002", "A-1001"},
		"field": map[string]any{"key": "endDate", "value": map[string]any{"date": "2026-06-20"}}}

	left := decoded[bulk.Result](t, priya.Do(t, http.MethodPost, previewPath, body), http.StatusOK)
	body["restartApproval"] = true
	restarted := decoded[bulk.Result](t, priya.Do(t, http.MethodPost, runPath, body), http.StatusCreated)

	if len(left.Reapproval) != 1 || left.Reapproval[0].RecordID != "A-1002" || outcomesOf(left)["A-1002"] == "done: " {
		t.Errorf("leave unchanged: reapproval %+v, outcomes %v", left.Reapproval, outcomesOf(left))
	}
	if got := outcomesOf(restarted); got["A-1002"] != "done: " {
		t.Errorf("restart approval: outcomes %v", got)
	}
}

func TestRun_Submit_Contract_012FR012_FR013(t *testing.T) {
	t.Parallel()
	priya := apitest.New(t).SignIn(t, priyasEmail)

	result := decoded[bulk.Result](t, priya.Do(t, http.MethodPost, previewPath,
		map[string]any{"action": "submit", "recordIds": []string{"C-1001", "A-1002", "A-1004", "A-1005"}}), http.StatusOK)

	got := outcomesOf(result)
	want := map[string]string{"C-1001": "skipped: Only activities go through approval", "A-1002": "skipped: Already approved",
		"A-1004": "skipped: Already submitted"}
	for id, outcome := range want {
		if got[id] != outcome {
			t.Errorf("%s = %q, want %q", id, got[id], outcome)
		}
	}
	if result.Verb != "submitted" || got["A-1005"] == "" {
		t.Errorf("verb %s, declined activity %q", result.Verb, got["A-1005"])
	}
}

func TestTargets_Contract_012FR016(t *testing.T) {
	t.Parallel()
	alex := apitest.New(t).SignIn(t, alexsEmail)

	page := decoded[bulk.TargetPage](t, alex.Do(t, http.MethodPost, "/api/v1/bulk-actions/targets",
		map[string]any{"action": "copy", "recordIds": []string{"A-1001"}, "search": "events"}), http.StatusOK)
	mixed := decoded[bulk.TargetPage](t, alex.Do(t, http.MethodPost, "/api/v1/bulk-actions/targets",
		map[string]any{"action": "move", "recordIds": []string{"A-1001", "A-1008"}}), http.StatusOK)

	paths := []string{}
	for _, target := range page.Items {
		paths = append(paths, target.Path)
	}
	if diff := cmp.Diff([]string{"FY2026 Lubricants Europe › Rimula fleet › Events", "FY2026 Lubricants Europe › Rimula fleet › Retail events"},
		paths); diff != "" {
		t.Errorf("copy targets (-want +got):\n%s", diff)
	}
	if mixed.Note == nil || *mixed.Note != "Move works within one plan — select records of one plan" || len(mixed.Items) != 0 {
		t.Errorf("targets of two plans = %+v", mixed)
	}
}

func TestRun_CopyMoveDeleteUndo_Contract_012FR014_FR020(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	copied := decoded[bulk.Result](t, priya.Do(t, http.MethodPost, runPath,
		map[string]any{"action": "copy", "recordIds": []string{"A-1001"}, "targetId": "I-1001"}), http.StatusCreated)
	moved := decoded[bulk.Result](t, priya.Do(t, http.MethodPost, runPath,
		map[string]any{"action": "move", "recordIds": []string{"A-1003"}, "targetId": "I-1001"}), http.StatusCreated)
	deleted := decoded[bulk.Result](t, priya.Do(t, http.MethodPost, runPath,
		map[string]any{"action": "delete", "recordIds": []string{"I-1003", "A-1001", copied.Outcomes[0].NewRecordID}}), http.StatusCreated)

	if copied.Outcomes[0].Outcome != "done" || moved.Outcomes[0].Outcome != "done" {
		t.Fatalf("copy %v, move %v", outcomesOf(copied), outcomesOf(moved))
	}
	if len(deleted.RecycleBinEntries) != 3 || deleted.DeleteSummary == nil || deleted.UndoAvailableUntil == nil {
		t.Fatalf("delete = %v %+v", outcomesOf(deleted), deleted.DeleteSummary)
	}
	undone := decoded[bulk.UndoResult](t, priya.Do(t, http.MethodPost, runPath+"/"+*deleted.ID+"/undo", nil), http.StatusOK)
	if len(undone.Restored) != 3 {
		t.Errorf("restored %d records, want 3", len(undone.Restored))
	}
	again := priya.Do(t, http.MethodPost, runPath+"/"+*deleted.ID+"/undo", nil)
	if body := decoded[apiError](t, again, http.StatusConflict); body.Error.Code != "already_resolved" {
		t.Errorf("a second undo = %+v", body)
	}
	late := decoded[bulk.Result](t, priya.Do(t, http.MethodPost, runPath,
		map[string]any{"action": "delete", "recordIds": []string{"A-1001"}}), http.StatusCreated)
	server.Advance(3 * time.Minute)
	if body := decoded[apiError](t, priya.Do(t, http.MethodPost, runPath+"/"+*late.ID+"/undo", nil), http.StatusConflict); body.Error.Code != "undo_expired" {
		t.Errorf("undo after 3 minutes = %+v", body)
	}
}
