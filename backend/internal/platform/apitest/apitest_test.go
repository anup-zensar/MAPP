package apitest_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi-validator/schema_validation"

	"mapp/api"
	"mapp/internal/platform/apitest"
)

// A handler test cannot pass with a request the web app could never send (research R11).
func TestDo_ReportsARequestThatBreaksTheContract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	recorder := &failureRecorder{TB: t}
	notAnEmail := map[string]any{"email": 42, "password": "mapp"}

	server.Do(recorder, http.MethodPost, "/api/v1/auth/sign-in", notAnEmail)
	if len(recorder.failures) == 0 {
		t.Error("a sign-in with a number as e-mail was not reported")
	}

	recorder.failures = nil
	server.Do(recorder, http.MethodPost, "/api/v1/auth/sign-in", notAnEmail, apitest.SkipRequestValidation())
	if len(recorder.failures) != 0 {
		t.Errorf("SkipRequestValidation() still reported: %q", recorder.failures)
	}
}

func TestDo_ReportsAPathThatIsNotInTheContract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	recorder := &failureRecorder{TB: t}

	server.Do(recorder, http.MethodGet, "/api/v1/nowhere", nil)
	if len(recorder.failures) == 0 {
		t.Error("a path that is not in the contract was not reported")
	}

	recorder.failures = nil
	server.Do(recorder, http.MethodGet, "/api/v1/nowhere", nil, apitest.SkipValidation())
	if len(recorder.failures) != 0 {
		t.Errorf("SkipValidation() still reported: %q", recorder.failures)
	}
}

func TestContract_IsAValidOpenAPIDocument(t *testing.T) {
	t.Parallel()
	document, err := libopenapi.NewDocument(api.OpenAPI)
	if err != nil {
		t.Fatalf("reading the contract: %v", err)
	}

	if valid, findings := schema_validation.ValidateOpenAPIDocument(document); !valid {
		for _, finding := range findings {
			t.Errorf("contract: %v", finding)
		}
	}
}

// failureRecorder replaces only Errorf of the real test, so a test can check what the helper reports.
type failureRecorder struct {
	testing.TB
	failures []string
}

func (r *failureRecorder) Errorf(format string, args ...any) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

// The role applies to Sara's next request, without signing in again (feature 014 D-228).
func TestServer_GrantRole(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	sara := server.SignIn(t, "sara.novak@shell.com")
	before := sara.Do(t, http.MethodGet, "/api/v1/access-rules", nil)
	if before.Status != http.StatusForbidden {
		t.Fatalf("a Viewer reading the access rules = %d %s, want 403", before.Status, before.Body)
	}

	server.GrantRole(t, "sara.novak@shell.com", "administrator", "all")
	after := sara.Do(t, http.MethodGet, "/api/v1/access-rules", nil)
	if after.Status != http.StatusOK {
		t.Errorf("after GrantRole = %d %s, want 200", after.Status, after.Body)
	}
	var me struct {
		Access struct {
			Roles []struct {
				Role  struct{ Name string } `json:"role"`
				Scope struct{ Text string } `json:"scope"`
			} `json:"roles"`
		} `json:"access"`
	}
	sara.Do(t, http.MethodGet, "/api/v1/me", nil).Decode(t, &me)
	got := fmt.Sprint(me.Access.Roles)
	if got != "[{{Administrator} {All markets}} {{Viewer} {My teams}}]" {
		t.Errorf("Sara's roles = %s", got)
	}
}
