package overspend_test

import (
	"net/http"
	"slices"
	"testing"

	"mapp/internal/campaign"
	"mapp/internal/history"
	"mapp/internal/overspend"
	"mapp/internal/platform/apitest"
)

const configurationPath = "/api/v1/overspend-configuration"

func TestConfiguration_ReadAndChange_Contract_037_FR006(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya, alex := server.SignIn(t, priyasEmail), server.SignIn(t, alexsEmail)

	asPriya := decoded[overspend.ConfigurationView](t, priya.Do(t, http.MethodGet, configurationPath, nil), http.StatusOK)
	forbidden := priya.Do(t, http.MethodPut, configurationPath, map[string]any{"version": 1, "defaultMode": "refuse"})
	invalid := alex.Do(t, http.MethodPut, configurationPath, map[string]any{"version": 1, "defaultMode": "block"})
	saved := decoded[overspend.ConfigurationView](t, alex.Do(t, http.MethodPut, configurationPath,
		map[string]any{"version": 1, "defaultMode": "refuse"}), http.StatusOK)
	stale := alex.Do(t, http.MethodPut, configurationPath, map[string]any{"version": 1, "defaultMode": "warn"})

	if asPriya.DefaultMode != overspend.Warn || asPriya.CanChange.Available || asPriya.CanChange.Reason == "" {
		t.Errorf("037 FR-006 as Priya = %+v, want Warn, read-only with the reason", asPriya)
	}
	if got := []int{forbidden.Status, invalid.Status, stale.Status}; !slices.Equal(got, []int{403, 400, 409}) {
		t.Errorf("037 FR-006 refusals = %v, want 403, 400, 409", got)
	}
	if saved.DefaultMode != overspend.Refuse || saved.Version != 2 || saved.UpdatedBy.Name != "Alex Morgan" {
		t.Errorf("037 FR-006 saved = %+v", saved)
	}
}

// configurationAnswer reads the answer with its last change as pointers, null while never saved.
type configurationAnswer struct {
	DefaultMode             overspend.Mode
	ApprovalOverPlannedCost overspend.Mode
	Version                 int
	UpdatedAt               *string
	UpdatedBy               *struct{ Name string }
}

func TestApprovalOverPlannedCost_Contract_038_FR018_FR057_M18_L16(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya, alex := server.SignIn(t, priyasEmail), server.SignIn(t, alexsEmail)
	refuse := map[string]any{"version": 1, "defaultMode": "warn", "approvalOverPlannedCost": "refuse"}

	fresh := decoded[configurationAnswer](t, priya.Do(t, http.MethodGet, configurationPath, nil), http.StatusOK)
	forbidden := priya.Do(t, http.MethodPut, configurationPath, refuse)
	invalid := alex.Do(t, http.MethodPut, configurationPath,
		map[string]any{"version": 1, "defaultMode": "warn", "approvalOverPlannedCost": "block"})
	saved := decoded[configurationAnswer](t, alex.Do(t, http.MethodPut, configurationPath, refuse), http.StatusOK)
	kept := decoded[configurationAnswer](t, alex.Do(t, http.MethodPut, configurationPath,
		map[string]any{"version": 2, "defaultMode": "refuse"}), http.StatusOK)
	stale := alex.Do(t, http.MethodPut, configurationPath, refuse)
	var log history.LogPage
	alex.Do(t, http.MethodGet, "/api/v1/activity-log?recordType=overspendConfiguration&recordId=overspend", nil).Decode(t, &log)

	if fresh.ApprovalOverPlannedCost != overspend.Warn || fresh.UpdatedAt != nil || fresh.UpdatedBy != nil {
		t.Errorf("038 FR-018 FR-057 never saved = %+v, want Warn and no last change", fresh)
	}
	if got := []int{forbidden.Status, invalid.Status, stale.Status}; !slices.Equal(got, []int{403, 400, 409}) {
		t.Errorf("038 FR-018 refusals = %v, want 403, 400, 409", got)
	}
	if saved.ApprovalOverPlannedCost != overspend.Refuse || saved.Version != 2 || saved.UpdatedBy == nil ||
		kept.ApprovalOverPlannedCost != overspend.Refuse || kept.DefaultMode != overspend.Refuse {
		t.Errorf("038 FR-018 saved = %+v, then %+v", saved, kept)
	}
	if got := logTexts(log); !slices.Contains(got, "Approval over planned cost: Warn → Refuse") {
		t.Errorf("038 FR-018 log = %v", got)
	}
}

func logTexts(page history.LogPage) []string {
	texts := []string{}
	for _, e := range page.Items {
		for _, c := range e.Changes {
			texts = append(texts, c.Label+": "+c.OldText+" → "+c.NewText)
		}
	}
	return texts
}

func TestNewBudgetsStartWithTheDefault_CopiesKeepTheirs_Contract_037_FR007(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya, alex := server.SignIn(t, priyasEmail), server.SignIn(t, alexsEmail)
	before := decoded[campaign.CampaignView](t, priya.Do(t, http.MethodGet, "/api/v1/campaigns/C-1002", nil), http.StatusOK)
	decoded[overspend.ConfigurationView](t, alex.Do(t, http.MethodPut, configurationPath,
		map[string]any{"version": 1, "defaultMode": "refuse"}), http.StatusOK)

	fresh := decoded[campaign.CampaignView](t, priya.Do(t, http.MethodPost, "/api/v1/plans/P-1001/campaigns",
		map[string]any{"name": "Started on Refuse"}), http.StatusCreated)
	chosen := decoded[campaign.CampaignView](t, priya.Do(t, http.MethodPost, "/api/v1/plans/P-1001/campaigns",
		map[string]any{"name": "Chosen Warn", "overspend": "warn"}), http.StatusCreated)
	copied := decoded[campaign.CampaignView](t, priya.Do(t, http.MethodPost, "/api/v1/campaigns/C-1002/copy",
		map[string]any{"planId": "P-1001", "name": "Rimula copy"}), http.StatusCreated)
	after := decoded[campaign.CampaignView](t, priya.Do(t, http.MethodGet, "/api/v1/campaigns/C-1002", nil), http.StatusOK)

	got := []overspend.Mode{fresh.Overspend.Mode, chosen.Overspend.Mode, copied.Overspend.Mode, after.Overspend.Mode}
	want := []overspend.Mode{overspend.Refuse, overspend.Warn, before.Overspend.Mode, before.Overspend.Mode}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("037 FR-007 new, chosen, copy, existing = %v, want %v", got, want)
			break
		}
	}
}
