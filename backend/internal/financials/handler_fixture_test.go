package financials_test

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"

	"mapp/internal/platform/apitest"
	"mapp/internal/platform/money"
)

// The contract tests of feature 015: every operation on the seed's demo data (data model §6), checked
// against backend/api/openapi.yaml by apitest. The test server's clock is 12 Mar 2026 09:00 UTC, so the
// current year is 2026.

// The demo people's e-mail addresses (demo-users-and-data.md).
const (
	alexsEmail  = "alex.morgan@shell.com"
	priyasEmail = "priya.raman@shell.com"
)

// get sends GET path as client, checks that it answers 200 and decodes the answer into a T.
func get[T any](t *testing.T, client *apitest.Client, path string) T {
	t.Helper()
	response := client.Do(t, http.MethodGet, path, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET %s = %d %s, want 200", path, response.Status, response.Body)
	}
	var v T
	response.Decode(t, &v)
	return v
}

// apiError is the inner object of an error answer.
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Fields  []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
	} `json:"fields"`
}

// refused checks that the answer has status and code and returns its body.
func refused(t *testing.T, response *apitest.Response, status int, code string) apiError {
	t.Helper()
	if response.Status != status {
		t.Fatalf("status %d %s, want %d %s", response.Status, response.Body, status, code)
	}
	var body struct {
		Error apiError `json:"error"`
	}
	response.Decode(t, &body)
	if body.Error.Code != code {
		t.Fatalf("code %q (%s), want %q", body.Error.Code, body.Error.Message, code)
	}
	return body.Error
}

// fieldRefused checks a 400 validation_failed on field with message (any message when "").
func fieldRefused(t *testing.T, response *apitest.Response, field, message string) {
	t.Helper()
	body := refused(t, response, http.StatusBadRequest, "validation_failed")
	if len(body.Fields) != 1 || body.Fields[0].Field != field || (message != "" && body.Fields[0].Message != message) {
		t.Errorf("fields = %+v, want %s: %q", body.Fields, field, message)
	}
}

// dollars is text as US dollars.
func dollars(t *testing.T, text string) money.Money {
	t.Helper()
	amount, err := money.ParseAmount(text)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", text, err)
	}
	return money.USD(amount)
}

// csvFile is a CSV file as the body of an import: lines joined with line breaks, as base64.
func csvFile(name string, lines ...string) map[string]string {
	content := strings.Join(lines, "\n") + "\n"
	return map[string]string{"fileName": name, "content": base64.StdEncoding.EncodeToString([]byte(content))}
}

// extractHeader is the header row of the simulated GSAP extract.
const extractHeader = "type,document_number,line,po_number,po_line,posting_date,accrual_month,amount,currency," +
	"vendor_code,vendor_name,description,swbs_code,tracking_key,line_item_id"
