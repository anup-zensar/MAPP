package approval_test

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/approval"
	"mapp/internal/platform/apitest"
)

func TestResubmitRoute(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-017 FR-018 FR-019 re-submitting the submitted Trade magazine ads", func(t *testing.T) {
		t.Parallel()
		server := apitest.New(t)
		priya := server.SignIn(t, priyasEmail)
		before := decodeApproval(t, priya.Do(t, http.MethodGet, "/api/v1/activities/"+tradeMagazineAds+"/approval", nil))
		var submission approval.Submission
		priya.Do(t, http.MethodGet, "/api/v1/activities/"+tradeMagazineAds+"/approval/submission", nil).Decode(t, &submission)

		after := decodeApproval(t, priya.Do(t, http.MethodPost, "/api/v1/activities/"+tradeMagazineAds+"/approval/submit",
			map[string]any{"version": before.ActivityVersion}))

		got := []any{before.Actions.Resubmit.Available, before.Actions.Recall.Available, submission.Kind, submission.Replaces,
			after.Status, after.Rounds[0].Number, after.Rounds[0].Kind, after.Rounds[0].Outcome, after.Rounds[1].Outcome}
		want := []any{true, true, "resubmit", &approval.ReplacedRound{RoundNumber: 2, Level: 1, LevelLabel: "Level 1"},
			"submitted", 3, approval.KindActivity, approval.OutcomeOpen, approval.OutcomeSuperseded}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("re-submission (-want +got):\n%s", diff)
		}
		daniel := server.SignIn(t, danielsEmail)
		stale := daniel.Do(t, http.MethodPost, "/api/v1/activities/"+tradeMagazineAds+"/approval/approve",
			map[string]any{"roundId": before.Rounds[0].ID, "level": 1})
		assertError(t, stale, http.StatusConflict, "version_conflict", "The submission was re-submitted by Priya Raman")
	})
}

func TestWorkflowObjectTypeRoutes(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-004 FR-005 FR-011 object types through the API", func(t *testing.T) {
		t.Parallel()
		server := apitest.New(t)
		alex := server.SignIn(t, alexsEmail)

		created := alex.Do(t, http.MethodPost, "/api/v1/approval-workflows", map[string]any{
			"name": "Purchase request approval", "objectType": "purchaseRequest",
			"levels": []map[string]any{{"name": "", "approverIds": []string{}, "approverType": "recordField", "recordField": "moaApprover"}},
		})
		var view approval.WorkflowView
		created.Decode(t, &view)
		if created.Status != http.StatusCreated || view.ObjectType != approval.KindPurchaseRequest ||
			view.Levels[0].RecordField == nil || view.Levels[0].RecordField.Label != "MoA approver" {
			t.Fatalf("created = %d %+v, want a purchase-request workflow approved by the MoA approver", created.Status, view)
		}
		refused := alex.Do(t, http.MethodPatch, "/api/v1/approval-workflows/WF-1001", map[string]any{"objectType": "purchaseRequest", "version": 1})
		assertError(t, refused, http.StatusBadRequest, "validation_failed", "Some fields need your attention.")

		var options approval.RuleOptions
		alex.Do(t, http.MethodGet, "/api/v1/approval-rule-options", nil).Decode(t, &options)
		if len(options.ObjectTypes) != 3 || len(options.RecordFields) != 2 || options.RecordFields[0].ID != approval.RecordFieldMoAApprover {
			t.Errorf("options = %+v / %+v, want three object types and the MoA approver field", options.ObjectTypes, options.RecordFields)
		}
	})
}

func TestRowsAndNotificationsCarrySubject(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-026 FR-028 rows and notifications name their record type", func(t *testing.T) {
		t.Parallel()
		server := apitest.New(t)
		daniel := server.SignIn(t, danielsEmail)

		rows := decodeRows(t, daniel.Do(t, http.MethodGet, "/api/v1/approvals?tab=open", nil))
		var page approval.NotificationPage
		daniel.Do(t, http.MethodGet, "/api/v1/notifications", nil).Decode(t, &page)

		for _, row := range rows.Items {
			if row.Subject.Kind != approval.KindActivity || row.Subject.Label != "Activity" || row.Activity == nil || row.Subject.ID != row.Activity.ID {
				t.Errorf("row = %+v, want an activity row", row.Subject)
			}
		}
		if len(page.Items) == 0 || page.Items[0].Subject.Kind != approval.KindActivity || page.Items[0].Activity == nil {
			t.Errorf("notifications = %+v, want activity notifications with their subject", page.Items)
		}
	})
}
