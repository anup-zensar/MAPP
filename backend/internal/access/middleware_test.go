package access

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/logging"
	"mapp/internal/platform/web"
)

// serve sends one request for the operation pattern as the person a, signed in as auth.RequireSignIn does,
// through Authorize to next, and returns the answer.
func (f *fixture) serve(t *testing.T, pattern string, a actor.Actor, next gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	method, path, _ := strings.Cut(pattern, " ")
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	signIn := func(c *gin.Context) {
		ctx := logging.WithAttrs(actor.With(c.Request.Context(), a), slog.String("actor_id", a.ID))
		c.Request = c.Request.WithContext(ctx)
	}
	engine.Handle(method, ginPath.Replace(path), signIn, f.service.Authorize, next)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), method, "http://127.0.0.1"+path, nil))
	return w
}

// ginPath turns the contract's {name} parameters into Gin's :name.
var ginPath = strings.NewReplacer("{", ":", "}", "")

// reached is a handler that records that it ran and answers 204.
type reached struct {
	ran    bool
	grants authz.Grants
	found  bool
}

func (h *reached) serve(c *gin.Context) {
	h.ran = true
	h.grants, h.found = authz.From(c.Request.Context())
	c.Status(http.StatusNoContent)
}

// The middleware puts the person's Grants in the request's context for the services (FR-009).
func TestAuthorize_GrantsInContext_FR009(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	var h reached

	w := f.serve(t, "GET /api/v1/plans", priya, h.serve)

	if w.Code != http.StatusNoContent || !h.ran || !h.found || h.grants.PersonID != priya.ID ||
		!h.grants.InScope(authz.PlansEdit, planOf(lubricants.ID)) {
		t.Errorf("answer %d, handler ran %v with grants %+v (found %v)", w.Code, h.ran, h.grants, h.found)
	}
}

// A Need operation without the permission is refused with 403 forbidden and the role message before the
// handler runs (FR-009, FR-010, message 2).
func TestAuthorize_NeedRefused403_FR010(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	var h reached

	w := f.serve(t, "POST /api/v1/models", daniel, h.serve)

	if h.ran || w.Code != http.StatusForbidden {
		t.Fatalf("answer %d, handler ran %v; want 403 before the handler", w.Code, h.ran)
	}
	want := `{"error":{"code":"forbidden","message":"Changing Settings isn't part of your roles (Approver, Planner). ` +
		`Ask an administrator if you need it.","fields":[]}}`
	if got := strings.TrimSpace(w.Body.String()); got != want {
		t.Errorf("body = %s\nwant %s", got, want)
	}

	var admin reached
	if f.serve(t, "POST /api/v1/models", alex, admin.serve); !admin.ran {
		t.Error("the administrator was refused")
	}
}

// An InPlan operation is refused when no role includes the permission anywhere; a person who holds it in
// some scope passes on to the service, which checks the record's plan (FR-009).
func TestAuthorize_InPlanPreCheck_FR009(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	var viewer, planner reached

	refused := f.serve(t, "PATCH /api/v1/activities/{activityId}", sara, viewer.serve)
	f.serve(t, "PATCH /api/v1/activities/{activityId}", mei, planner.serve)

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
	t.Parallel()
	f := newFixture(t)
	if _, err := f.service.SetPersonRoles(t.Context(), alex, sara.ID, PersonRolesUpdate{Version: version(1), Roles: []AssignmentInput{}}); err != nil {
		t.Fatalf("SetPersonRoles: %v", err)
	}
	for _, pattern := range []string{"GET /api/v1/me", "GET /api/v1/plans/{planId}", "POST /api/v1/activities/{activityId}/approval/approve"} {
		var h reached
		if w := f.serve(t, pattern, sara, h.serve); !h.ran || w.Code != http.StatusNoContent {
			t.Errorf("%s without a role = %d, handler ran %v", pattern, w.Code, h.ran)
		}
	}
}

// An operation without an access rule is refused, whoever asks (fail closed, FR-010).
func TestAuthorize_NoRuleRefused_FR010(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	var h reached

	w := f.serve(t, "GET /api/v1/not-declared", alex, h.serve)

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
	slog.SetDefault(logging.New(&out, slog.LevelInfo))
	t.Cleanup(func() { slog.SetDefault(previous) })

	f.serve(t, "DELETE /api/v1/models/{modelId}", sara, (&reached{}).serve)
	f.serve(t, "DELETE /api/v1/plans/{planId}", priya, func(c *gin.Context) {
		web.RespondError(c, errs.Forbidden("Deleting plans isn't part of your role.")) // refused by the service
	})

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("technical log =\n%s\nwant two lines", out.String())
	}
	for i, want := range []map[string]string{
		{"operation": "DELETE /api/v1/models/{modelId}", "actor_id": "U-1006", "code": "forbidden"},
		{"operation": "DELETE /api/v1/plans/{planId}", "actor_id": "U-1002", "code": "forbidden"},
	} {
		var record map[string]any
		if err := json.Unmarshal([]byte(lines[i]), &record); err != nil {
			t.Fatalf("log line %d is not JSON: %s", i+1, lines[i])
		}
		for key, value := range want {
			if record[key] != value {
				t.Errorf("log line %d: %s = %v, want %s", i+1, key, record[key], value)
			}
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
