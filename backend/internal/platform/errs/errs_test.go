package errs_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
)

var (
	changedAt = time.Date(2026, 3, 12, 14, 32, 0, 0, time.UTC)
	priya     = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
)

// Every constructor gives its error the HTTP status, the code and the plain-language message of the
// plan's "User-visible messages" table (FR-020).
func TestConstructors_StatusCodeAndMessage_FR020(t *testing.T) {
	tests := []struct {
		name    string
		err     *errs.Error
		status  int
		code    string
		message string
	}{
		{
			"Validation", errs.Validation(errs.FieldError{Field: "name", Message: "Enter a name."}),
			http.StatusBadRequest, errs.CodeValidationFailed, "Some fields need your attention.",
		},
		{
			"BadRequest", errs.BadRequest("NorthStar couldn't read this request. Reload the page and try again."),
			http.StatusBadRequest, errs.CodeBadRequest, "NorthStar couldn't read this request. Reload the page and try again.",
		},
		{
			"SignInRequired", errs.SignInRequired(),
			http.StatusUnauthorized, errs.CodeSignInRequired, "Please sign in to continue.",
		},
		{
			"SessionExpired", errs.SessionExpired(),
			http.StatusUnauthorized, errs.CodeSessionExpired, "Please sign in again",
		},
		{
			"IncorrectCredentials", errs.IncorrectCredentials(),
			http.StatusUnauthorized, errs.CodeIncorrectCredentials, "Incorrect e-mail or password.",
		},
		{
			"NotFound", errs.NotFound("This item is not in the recycle bin."),
			http.StatusNotFound, errs.CodeNotFound, "This item is not in the recycle bin.",
		},
		{
			"VersionConflict", errs.VersionConflict("sample folder", "Daniel Okoro", changedAt),
			http.StatusConflict, errs.CodeVersionConflict, "Not saved — Daniel Okoro changed this sample folder",
		},
		{
			"RestoreParentFirst", errs.RestoreParentFirst(errs.RecordRef{Type: "sampleFolder", ID: "SF-1001", Name: "Ideas"}),
			http.StatusConflict, errs.CodeRestoreParentFirst, "Restore 'Ideas' first",
		},
		{
			"UndoExpired", errs.UndoExpired(),
			http.StatusConflict, errs.CodeUndoExpired, "Undo is no longer available — restore it from the recycle bin",
		},
		{
			"AlreadyResolved", errs.AlreadyResolved("Ideas", errs.Resolution{How: "restored", By: priya, At: changedAt}),
			http.StatusConflict, errs.CodeAlreadyResolved, "'Ideas' is no longer in the recycle bin.",
		},
		{
			"NameTaken", errs.NameTaken("Can't undo — there is now another sample folder called 'Ideas'. Restore it from the recycle bin instead."),
			http.StatusConflict, errs.CodeNameTaken, "Can't undo — there is now another sample folder called 'Ideas'. Restore it from the recycle bin instead.",
		},
		{
			"Internal", errs.Internal(),
			http.StatusInternalServerError, errs.CodeInternalError, "Something went wrong on our side. Please try again.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Status != tt.status {
				t.Errorf("Status = %d, want %d", tt.err.Status, tt.status)
			}
			if tt.err.Code != tt.code {
				t.Errorf("Code = %q, want %q", tt.err.Code, tt.code)
			}
			if tt.err.Message != tt.message {
				t.Errorf("Message = %q, want %q", tt.err.Message, tt.message)
			}
		})
	}
}

// The code strings are part of the API contract (research R9); the web app chooses its messages by them.
func TestCodes_MatchTheContract(t *testing.T) {
	codes := map[string]string{
		errs.CodeValidationFailed:     "validation_failed",
		errs.CodeBadRequest:           "bad_request",
		errs.CodeSignInRequired:       "sign_in_required",
		errs.CodeSessionExpired:       "session_expired",
		errs.CodeIncorrectCredentials: "incorrect_credentials",
		errs.CodeNotFound:             "not_found",
		errs.CodeVersionConflict:      "version_conflict",
		errs.CodeRestoreParentFirst:   "restore_parent_first",
		errs.CodeUndoExpired:          "undo_expired",
		errs.CodeAlreadyResolved:      "already_resolved",
		errs.CodeNameTaken:            "name_taken",
		errs.CodeInternalError:        "internal_error",
		errs.CodeInUse:                "in_use",
		errs.CodeCannotRestore:        "cannot_restore",
		errs.CodeNotAllowed:           "not_allowed",
		errs.CodeNotReady:             "not_ready",
		errs.CodeForbidden:            "forbidden",
	}
	for got, want := range codes {
		if got != want {
			t.Errorf("code constant = %q, want %q", got, want)
		}
	}
}

// Every error answer carries "fields", an empty list when there are no field problems, so the web app
// never has to check for a missing list (research R9, FR-020).
func TestError_FieldsEncodeAsEmptyList_FR020(t *testing.T) {
	tests := []struct {
		name string
		err  *errs.Error
		want string
	}{
		{
			"no fields", errs.NotFound("This page or record doesn't exist."),
			`{"code":"not_found","message":"This page or record doesn't exist.","fields":[]}`,
		},
		{
			"validation without fields", errs.Validation(),
			`{"code":"validation_failed","message":"Some fields need your attention.","fields":[]}`,
		},
		{
			"validation with fields", errs.Validation(errs.FieldError{Field: "email", Message: "Enter your e-mail"}),
			`{"code":"validation_failed","message":"Some fields need your attention.","fields":[{"field":"email","message":"Enter your e-mail"}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := encode(t, tt.err); got != tt.want {
				t.Errorf("JSON =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

// A version conflict names who changed the record and when; the server message has no time, because
// only the browser knows the user's time zone (FR-029, research R5).
func TestVersionConflict_NamesWhoAndWhen_FR029(t *testing.T) {
	got := encode(t, errs.VersionConflict("sample folder", "Daniel Okoro", changedAt))
	want := `{"code":"version_conflict","message":"Not saved — Daniel Okoro changed this sample folder","fields":[],` +
		`"conflict":{"changedBy":"Daniel Okoro","changedAt":"2026-03-12T14:32:00Z"}}`
	if got != want {
		t.Errorf("JSON =\n%s\nwant\n%s", got, want)
	}
}

// A refused restore names the parent to restore first (FR-025).
func TestRestoreParentFirst_NamesTheParent_FR025(t *testing.T) {
	got := encode(t, errs.RestoreParentFirst(errs.RecordRef{Type: "sampleFolder", ID: "SF-1001", Name: "Ideas"}))
	want := `{"code":"restore_parent_first","message":"Restore 'Ideas' first","fields":[],` +
		`"parent":{"type":"sampleFolder","id":"SF-1001","name":"Ideas"}}`
	if got != want {
		t.Errorf("JSON =\n%s\nwant\n%s", got, want)
	}
}

// An entry that was already undone or restored says how, by whom and when (FR-025, FR-026).
func TestAlreadyResolved_SaysHowByWhomAndWhen_FR025(t *testing.T) {
	got := encode(t, errs.AlreadyResolved("Ideas", errs.Resolution{How: "undone", By: priya, At: changedAt}))
	want := `{"code":"already_resolved","message":"'Ideas' is no longer in the recycle bin.","fields":[],` +
		`"resolution":{"how":"undone","by":{"id":"U-1002","name":"Priya Raman"},"at":"2026-03-12T14:32:00Z"}}`
	if got != want {
		t.Errorf("JSON =\n%s\nwant\n%s", got, want)
	}
}

// Problems collects every field problem of a request so that all of them are reported at once (FR-020).
func TestProblems_ReportsEveryFieldAtOnce_FR020(t *testing.T) {
	var problems errs.Problems
	if err := problems.Err(); err != nil {
		t.Fatalf("Err() with no problems = %v, want nil", err)
	}

	problems.Add("email", "Enter your e-mail")
	problems.Add("password", "Enter the password")

	var e *errs.Error
	if !errors.As(problems.Err(), &e) {
		t.Fatalf("Err() = %v, want an *errs.Error", problems.Err())
	}
	want := `{"code":"validation_failed","message":"Some fields need your attention.","fields":[` +
		`{"field":"email","message":"Enter your e-mail"},{"field":"password","message":"Enter the password"}]}`
	if got := encode(t, e); got != want {
		t.Errorf("JSON =\n%s\nwant\n%s", got, want)
	}
}

// An error wrapped with %w on its way up (for example through a store transaction) is still found, so
// it keeps its status (research R9).
func TestError_FoundThroughWrapping(t *testing.T) {
	wrapped := fmt.Errorf("deleting folder: %w", errs.NotFound("This sample folder no longer exists — someone may have deleted it."))

	var e *errs.Error
	if !errors.As(wrapped, &e) {
		t.Fatalf("errors.As did not find the *errs.Error in %v", wrapped)
	}
	if e.Status != http.StatusNotFound || e.Code != errs.CodeNotFound {
		t.Errorf("found %d %s, want 404 not_found", e.Status, e.Code)
	}
}

// The error text (for logs and test failures) shows the code and the message.
func TestError_Text(t *testing.T) {
	got := errs.SessionExpired().Error()
	if want := "session_expired: Please sign in again"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// encode returns the JSON the API would send for err.
func encode(t *testing.T, err *errs.Error) string {
	t.Helper()
	body, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("json.Marshal: %v", marshalErr)
	}
	return string(body)
}

// The Plan module's refusals are 409 conflicts with the message the caller gives; not_ready lists every
// missing field grouped by record, and the other errors leave "missing" out (D-043, D-193).
func TestPlanModuleErrors_409_D043(t *testing.T) {
	tests := []struct {
		err  *errs.Error
		code string
	}{
		{errs.InUse("'Lubricants 1.0' is in use (used by 2 plans) — set it inactive instead."), errs.CodeInUse},
		{errs.CannotRestore("Can't restore — there is now another currency with the code 'JPY'."), errs.CodeCannotRestore},
		{errs.NotAllowed("You can't deactivate yourself — ask someone else to do it."), errs.CodeNotAllowed},
	}
	for _, tt := range tests {
		if tt.err.Status != http.StatusConflict || tt.err.Code != tt.code || tt.err.Missing != nil {
			t.Errorf("%s: status %d, code %q, missing %v", tt.code, tt.err.Status, tt.err.Code, tt.err.Missing)
		}
		want := fmt.Sprintf(`{"code":%q,"message":%q,"fields":[]}`, tt.code, tt.err.Message)
		if got := encode(t, tt.err); got != want {
			t.Errorf("JSON =\n%s\nwant\n%s", got, want)
		}
	}

	notReady := errs.NotReady("Publish needs 2 fields", []errs.ReadinessGroup{{
		Record: errs.RecordRef{Type: "plan", ID: "P-1002", Name: "FY2026 Mobility China"},
		Fields: []errs.MissingField{
			{Field: "attributes.SA-1001", Label: "Business unit"},
			{Field: "attributes.SA-1011", Label: "Country / OU", Reason: "Shares add up to 90.00 %"},
		},
	}})
	want := `{"code":"not_ready","message":"Publish needs 2 fields","fields":[],"missing":[{"record":{"type":"plan","id":"P-1002","name":"FY2026 Mobility China"},"fields":[{"field":"attributes.SA-1001","label":"Business unit"},{"field":"attributes.SA-1011","label":"Country / OU","reason":"Shares add up to 90.00 %"}]}]}`
	if notReady.Status != http.StatusConflict {
		t.Errorf("NotReady status = %d, want 409", notReady.Status)
	}
	if got := encode(t, notReady); got != want {
		t.Errorf("NotReady JSON =\n%s\nwant\n%s", got, want)
	}
}

// A refusal by role or scope is 403 forbidden with the message the caller gives and no field problems
// (014 FR-010, D-227).
func TestForbidden_403WithMessage_FR010(t *testing.T) {
	message := "Changing activities and line items isn't part of your role (Viewer). Ask an administrator if you need it."
	err := errs.Forbidden(message)

	if err.Status != http.StatusForbidden || err.Code != errs.CodeForbidden || err.Message != message {
		t.Errorf("Forbidden = %d %q %q, want 403 forbidden with the message", err.Status, err.Code, err.Message)
	}
	want := fmt.Sprintf(`{"code":"forbidden","message":%q,"fields":[]}`, message)
	if got := encode(t, err); got != want {
		t.Errorf("JSON =\n%s\nwant\n%s", got, want)
	}
}
