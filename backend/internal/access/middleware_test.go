package access

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/web"
)

// serve sends one request for the operation pattern as the person a through Authorize to next, and returns
// the answer.
func serve(t *testing.T, f *fixture, pattern string, a actor.Actor, next http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	method, path, _ := strings.Cut(pattern, " ")
	r := httptest.NewRequest(method, "http://127.0.0.1"+path, nil)
	r.Pattern = pattern
	r = r.WithContext(actor.With(r.Context(), a))
	w := httptest.NewRecorder()
	f.service.Authorize(next)(w, r)
	return w
}

// reached is a handler that records that it ran and answers 204.
type reached struct {
	ran    bool
	grants authz.Grants
	found  bool
}

func (h *reached) serve(w http.ResponseWriter, r *http.Request) {
	h.ran = true
	h.grants, h.found = authz.From(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

// The middleware puts the person's Grants in the request's context for the services (FR-009).
func TestAuthorize_GrantsInContext_FR009(t *testing.T) {
	f := newFixture(t)
	var h reached

	w := serve(t, f, "GET /api/v1/plans", priya, h.serve)

	if w.Code != http.StatusNoContent || !h.ran || !h.found || h.grants.PersonID != priya.ID ||
		!h.grants.InScope(authz.PlansEdit, planOf(lubricants.ID)) {
		t.Errorf("answer %d, handler ran %v with grants %+v (found %v)", w.Code, h.ran, h.grants, h.found)
	}
}

// A Need operation without the permission is refused with 403 forbidden and the role message before the
// handler runs (FR-009, FR-010, message 2).
func TestAuthorize_NeedRefused403_FR010(t *testing.T) {
	f := newFixture(t)
	var h reached

	w := serve(t, f, "POST /api/v1/models", daniel, h.serve)

	if h.ran || w.Code != http.StatusForbidden {
		t.Fatalf("answer %d, handler ran %v; want 403 before the handler", w.Code, h.ran)
	}
	want := `{"error":{"code":"forbidden","message":"Changing Settings isn't part of your roles (Approver, Planner). ` +
		`Ask an administrator if you need it.","fields":[]}}`
	if got := strings.TrimSpace(w.Body.String()); got != want {
		t.Errorf("body = %s\nwant %s", got, want)
	}

	var admin reached
	if serve(t, f, "POST /api/v1/models", alex, admin.serve); !admin.ran {
		t.Error("the administrator was refused")
	}
}

// An InPlan operation is refused when no role includes the permission anywhere; a person who holds it in
// some scope passes on to the service, which checks the record's plan (FR-009).
func TestAuthorize_InPlanPreCheck_FR009(t *testing.T) {
	f := newFixture(t)
	var viewer, planner reached

	refused := serve(t, f, "PATCH /api/v1/activities/{activityId}", sara, viewer.serve)
	serve(t, f, "PATCH /api/v1/activities/{activityId}", mei, planner.serve)

	if viewer.ran || refused.Code != http.StatusForbidden ||
		!strings.Contains(refused.Body.String(), "Changing activities and line items isn't part of your role (Viewer).") {
		t.Errorf("Sara = %d %s (handler ran %v), want 403 with the role message", refused.Code, refused.Body, viewer.ran)
	}
	if !planner.ran {
		t.Error("Mei, a Planner for Mobility China, did not reach the service")
	}
}

// View, Custom and SignedIn operations pass on to the service whatever the roles; a person without any role
// still reaches the operations open to everyone signed in (FR-009).
func TestAuthorize_ViewAndSignedInPass_FR009(t *testing.T) {
	f := newFixture(t)
	if _, err := f.service.SetPersonRoles(t.Context(), alex, sara.ID, PersonRolesUpdate{Version: version(1), Roles: []AssignmentInput{}}); err != nil {
		t.Fatalf("SetPersonRoles: %v", err)
	}
	for _, pattern := range []string{"GET /api/v1/me", "GET /api/v1/plans/{planId}", "POST /api/v1/activities/{activityId}/approval/approve"} {
		var h reached
		if w := serve(t, f, pattern, sara, h.serve); !h.ran || w.Code != http.StatusNoContent {
			t.Errorf("%s without a role = %d, handler ran %v", pattern, w.Code, h.ran)
		}
	}
}

// An operation without an access rule is refused, whoever asks (fail closed, FR-010).
func TestAuthorize_NoRuleRefused_FR010(t *testing.T) {
	f := newFixture(t)
	var h reached

	w := serve(t, f, "GET /api/v1/not-declared", alex, h.serve)

	if h.ran || w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), `"code":"forbidden"`) {
		t.Errorf("an operation without a rule = %d %s (handler ran %v), want 403", w.Code, w.Body, h.ran)
	}
}

// Every refusal — by the middleware or by a service — is written to the technical log with the operation,
// the person's ID and the code forbidden, never a name or an e-mail (FR-039, constitution VI). Refusals are
// not in the activity log (D-241).
func TestAuthorize_RefusalsInTheTechnicalLogOnly_FR039(t *testing.T) {
	f := newFixture(t)
	var out bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&out, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	serve(t, f, "DELETE /api/v1/models/{modelId}", sara, (&reached{}).serve)
	serve(t, f, "DELETE /api/v1/plans/{planId}", priya, func(w http.ResponseWriter, r *http.Request) {
		web.WriteError(w, r, errs.Forbidden("Deleting plans isn't part of your role.")) // refused by the service
	})

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("technical log =\n%s\nwant two lines", out.String())
	}
	for i, want := range []string{`operation="DELETE /api/v1/models/{modelId}" person=U-1006 code=forbidden`,
		`operation="DELETE /api/v1/plans/{planId}" person=U-1002 code=forbidden`} {
		if !strings.Contains(lines[i], want) {
			t.Errorf("log line %d = %s, want it to contain %s", i+1, lines[i], want)
		}
	}
	for _, personal := range []string{"Sara", "Novak", "Priya", "Raman", "@"} {
		if strings.Contains(out.String(), personal) {
			t.Errorf("the technical log names %q:\n%s", personal, out.String())
		}
	}
	if entries := f.log(t); len(entries) != 0 {
		t.Errorf("refusals wrote %d activity-log entries, want none", len(entries))
	}
}
