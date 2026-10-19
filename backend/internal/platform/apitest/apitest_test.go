package apitest_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"

	"northstar/api"
	"northstar/internal/platform/apitest"
)

// The helper checks every request against the contract, so a handler test cannot pass with a request
// the web app could never send (research R11).
func TestDo_ReportsARequestThatBreaksTheContract(t *testing.T) {
	server := apitest.New(t)
	recorder := &failureRecorder{TB: t}
	notAnEmail := map[string]any{"email": 42, "password": "northstar"}

	server.Do(recorder, http.MethodPost, "/api/auth/sign-in", notAnEmail)
	if len(recorder.failures) == 0 {
		t.Error("a sign-in with a number as e-mail was not reported")
	}

	recorder.failures = nil
	server.Do(recorder, http.MethodPost, "/api/auth/sign-in", notAnEmail, apitest.SkipRequestValidation())
	if len(recorder.failures) != 0 {
		t.Errorf("SkipRequestValidation() still reported: %q", recorder.failures)
	}
}

// Paths and methods that are not in the contract are reported, unless the test says it calls one on
// purpose.
func TestDo_ReportsAPathThatIsNotInTheContract(t *testing.T) {
	server := apitest.New(t)
	recorder := &failureRecorder{TB: t}

	server.Do(recorder, http.MethodGet, "/api/nowhere", nil)
	if len(recorder.failures) == 0 {
		t.Error("a path that is not in the contract was not reported")
	}

	recorder.failures = nil
	server.Do(recorder, http.MethodGet, "/api/nowhere", nil, apitest.SkipValidation())
	if len(recorder.failures) != 0 {
		t.Errorf("SkipValidation() still reported: %q", recorder.failures)
	}
}

func TestResponse_CookieAndDecode(t *testing.T) {
	response := &apitest.Response{
		Status: http.StatusOK,
		Header: http.Header{"Set-Cookie": {"northstar_session=abc123; Path=/; HttpOnly; SameSite=Strict"}},
		Body:   []byte(`{"status":"ok"}`),
	}

	cookie := response.Cookie("northstar_session")
	if cookie == nil || cookie.Value != "abc123" || !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("Cookie(northstar_session) = %+v, want abc123, HttpOnly, SameSite=Strict", cookie)
	}
	if other := response.Cookie("other"); other != nil {
		t.Errorf("Cookie(other) = %+v, want nil", other)
	}

	var body struct {
		Status string `json:"status"`
	}
	response.Decode(t, &body)
	if body.Status != "ok" {
		t.Errorf("Decode: status = %q, want ok", body.Status)
	}
}

// The contract the helper checks against is itself a valid OpenAPI document.
func TestContract_IsAValidOpenAPIDocument(t *testing.T) {
	document, err := libopenapi.NewDocument(api.OpenAPI)
	if err != nil {
		t.Fatalf("reading the contract: %v", err)
	}
	contract, problems := validator.NewValidator(document)
	if len(problems) > 0 {
		t.Fatalf("building the validator: %v", problems)
	}

	if valid, findings := contract.ValidateDocument(); !valid {
		for _, finding := range findings {
			t.Errorf("contract: %v", finding)
		}
	}
}

// failureRecorder stands in for the test's *testing.T so that a test can check what the helper reports.
// Only Errorf is replaced; everything else goes to the real test.
type failureRecorder struct {
	testing.TB
	failures []string
}

func (r *failureRecorder) Errorf(format string, args ...any) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}
