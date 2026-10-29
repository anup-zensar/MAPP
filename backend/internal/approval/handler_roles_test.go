package approval_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apitest"
)

// Approvals read people's roles from Settings > Roles and access: taking Daniel Okoro's Approver role away
// there stops him deciding, and giving it back lets him approve again.
func TestApprove_FollowsRolesChangedInSettings_017_FR017(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	daniel := server.SignIn(t, danielsEmail)
	roundID := decodeApproval(t, daniel.Do(t, http.MethodGet, "/api/v1/activities/"+tradeMagazineAds+"/approval", nil)).Rounds[0].ID
	decision := map[string]any{"roundId": roundID, "level": 1}
	planner := map[string]any{"roleId": "RL-1003", "scope": map[string]any{"type": "myTeams"}}
	approver := map[string]any{"roleId": "RL-1004", "scope": map[string]any{"type": "myTeams"}}

	setRoles(t, alex, 1, planner)
	refused := daniel.Do(t, http.MethodPost, "/api/v1/activities/"+tradeMagazineAds+"/approval/approve", decision)
	setRoles(t, alex, 2, planner, approver)
	approved := daniel.Do(t, http.MethodPost, "/api/v1/activities/"+tradeMagazineAds+"/approval/approve", decision)

	if refused.Status != http.StatusForbidden {
		t.Errorf("approve without an approving role = %d %s, want 403", refused.Status, refused.Body)
	}
	if appr := decodeApproval(t, approved); appr.Status != "approved" {
		t.Errorf("approve with the Approver role given back: status %s, want approved", appr.Status)
	}
}

func setRoles(t *testing.T, alex *apitest.Client, version int, roles ...map[string]any) {
	t.Helper()
	response := alex.Do(t, http.MethodPut, "/api/v1/people/U-1003/roles", map[string]any{"version": version, "roles": roles})
	if response.Status != http.StatusOK {
		t.Fatalf("set Daniel's roles = %d %s, want 200", response.Status, response.Body)
	}
}
