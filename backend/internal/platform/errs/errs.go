// Package errs defines MAPP's one error format (research R9).
//
// Every API failure is answered as JSON {"error": {"code", "message", "fields"}}, plus "conflict", "parent",
// "resolution" or "missing" inside "error" for the errors that need them. The message is written for users
// (the texts come from the plan's "User-visible messages" table); the code is for the web app's logic.
// Services return these errors, and web.WriteError turns them into HTTP answers.
package errs

import (
	"fmt"
	"net/http"
	"time"

	"mapp/internal/platform/actor"
)

// The error codes of the API contract (research R9). The web app chooses what to show by the code.
const (
	CodeValidationFailed     = "validation_failed"     // one or more fields are wrong; Fields lists all of them
	CodeBadRequest           = "bad_request"           // the request could not be read, or came from another site
	CodeSignInRequired       = "sign_in_required"      // no valid session
	CodeSessionExpired       = "session_expired"       // the session ended after 12 hours without activity
	CodeIncorrectCredentials = "incorrect_credentials" // sign-in failed
	CodeNotFound             = "not_found"             // unknown record or path
	CodeVersionConflict      = "version_conflict"      // saved from an out-of-date copy
	CodeRestoreParentFirst   = "restore_parent_first"  // the parent of a record to restore is missing
	CodeUndoExpired          = "undo_expired"          // Undo asked for too late
	CodeAlreadyResolved      = "already_resolved"      // the recycle-bin entry was already undone or restored
	CodeNameTaken            = "name_taken"            // Undo refused because the name is now taken
	CodeInternalError        = "internal_error"        // anything unexpected

	// Codes of the Plan module (features 002–008), added once by the shared foundation.
	CodeInUse         = "in_use"         // a record in use cannot be deleted; set it inactive instead (002, D-043)
	CodeCannotRestore = "cannot_restore" // a record cannot come back from the recycle bin, with the reason (002)
	CodeNotAllowed    = "not_allowed"    // the person may not do this, such as deactivate themselves (003)
	CodeNotReady      = "not_ready"      // needed fields are missing for publish or submit; Missing lists them (004)

	// Code of feature 014, access control: the person's roles do not allow the action (D-227).
	CodeForbidden = "forbidden"

	// CodeParentDeletedPermanently refuses a restore whose parent, or a record above it, was deleted
	// permanently; Parent names that record (011, D-182).
	CodeParentDeletedPermanently = "parent_deleted_permanently"
)

// internalMessage is the message of an unexpected failure; the details go only to the server log.
const internalMessage = "Something went wrong on our side. Please try again."

// Error is an API failure. Status is the HTTP status and is not part of the JSON.
type Error struct {
	Status     int          `json:"-"`
	Code       string       `json:"code"`
	Message    string       `json:"message"`
	Fields     []FieldError `json:"fields"` // never nil, so the JSON always has a list
	Conflict   *Conflict    `json:"conflict,omitempty"`
	Parent     *RecordRef   `json:"parent,omitempty"`
	Resolution *Resolution  `json:"resolution,omitempty"`
	// Missing lists every needed field that stops the action, grouped by record; only with not_ready.
	Missing []ReadinessGroup `json:"missing,omitempty"`
}

// Error returns the code and the message, for logs and test failures.
func (e *Error) Error() string {
	return e.Code + ": " + e.Message
}

// FieldError is the problem with one field of a request, shown next to that field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Conflict says who changed a record, and when, after the user opened it (version_conflict). The web app
// shows the time in the user's own time zone.
type Conflict struct {
	ChangedBy string    `json:"changedBy"`
	ChangedAt time.Time `json:"changedAt"`
}

// RecordRef points at a record: its kind, its ID and its name at the time.
type RecordRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// MissingField is a field that is still needed to publish or submit: the one MissingField of the Plan
// module (contract schema MissingField, decision D-193). settings.MissingField is this type.
type MissingField struct {
	Field  string `json:"field"`            // the form field, such as "ownerId" or "attributes.SA-1003"
	Label  string `json:"label"`            // the field's label, such as "Cluster"
	Reason string `json:"reason,omitempty"` // why a filled-in field still counts, such as a split below 100 %
}

// ReadinessGroup is the fields one record still needs before an action (contract schema ReadinessGroup):
// publishing a plan, submitting an activity.
type ReadinessGroup struct {
	Record RecordRef      `json:"record"`
	Fields []MissingField `json:"fields"`
}

// Resolution says how a recycle-bin entry left the bin ("undone" or "restored"), by whom and when.
type Resolution struct {
	How string      `json:"how"`
	By  actor.Actor `json:"by"`
	At  time.Time   `json:"at"`
}

// newError builds an Error with an empty list of field problems.
func newError(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message, Fields: []FieldError{}}
}

// Validation reports every field problem of a request at once (FR-020).
func Validation(fields ...FieldError) *Error {
	e := newError(http.StatusBadRequest, CodeValidationFailed, "Some fields need your attention.")
	e.Fields = append(e.Fields, fields...)
	return e
}

// BadRequest reports a request that could not be used, with a message for the user.
func BadRequest(message string) *Error {
	return newError(http.StatusBadRequest, CodeBadRequest, message)
}

// SignInRequired reports a request without a valid session.
func SignInRequired() *Error {
	return newError(http.StatusUnauthorized, CodeSignInRequired, "Please sign in to continue.")
}

// SessionExpired reports a session that ended after 12 hours without activity (FR-006).
func SessionExpired() *Error {
	return newError(http.StatusUnauthorized, CodeSessionExpired, "Please sign in again")
}

// IncorrectCredentials reports a failed sign-in without saying whether the e-mail or the password was
// wrong (FR-004).
func IncorrectCredentials() *Error {
	return newError(http.StatusUnauthorized, CodeIncorrectCredentials, "Incorrect e-mail or password.")
}

// NotFound reports an unknown record or path, with a message for the user.
func NotFound(message string) *Error {
	return newError(http.StatusNotFound, CodeNotFound, message)
}

// VersionConflict refuses a save made from an out-of-date copy of a record and says who changed it and
// when (FR-029). The message has no time: only the browser knows the user's time zone (research R5).
func VersionConflict(noun, changedBy string, changedAt time.Time) *Error {
	message := fmt.Sprintf("Not saved — %s changed this %s", changedBy, noun)
	e := newError(http.StatusConflict, CodeVersionConflict, message)
	e.Conflict = &Conflict{ChangedBy: changedBy, ChangedAt: changedAt}
	return e
}

// RestoreParentFirst refuses to restore a record whose parent no longer exists (FR-025).
func RestoreParentFirst(parent RecordRef) *Error {
	message := fmt.Sprintf("Restore '%s' first", parent.Name)
	e := newError(http.StatusConflict, CodeRestoreParentFirst, message)
	e.Parent = &parent
	return e
}

// UndoExpired refuses an Undo asked for too long after the delete (FR-026).
func UndoExpired() *Error {
	return newError(http.StatusConflict, CodeUndoExpired,
		"Undo is no longer available — restore it from the recycle bin")
}

// AlreadyResolved refuses to undo or restore a recycle-bin entry that has already left the bin, and says
// how, by whom and when.
func AlreadyResolved(name string, r Resolution) *Error {
	message := fmt.Sprintf("'%s' is no longer in the recycle bin.", name)
	e := newError(http.StatusConflict, CodeAlreadyResolved, message)
	e.Resolution = &r
	return e
}

// NameTaken refuses an Undo because someone has meanwhile taken the record's name, with a message for
// the user.
func NameTaken(message string) *Error {
	return newError(http.StatusConflict, CodeNameTaken, message)
}

// InUse refuses to delete a record that something uses, with a message for the user that names what uses
// it (409 in_use, D-043).
func InUse(message string) *Error {
	return newError(http.StatusConflict, CodeInUse, message)
}

// CannotRestore refuses to bring a record back from the recycle bin, with a message for the user that says
// why (409 cannot_restore).
func CannotRestore(message string) *Error {
	return newError(http.StatusConflict, CodeCannotRestore, message)
}

// NotAllowed refuses an action the person may not take, with a message for the user (409 not_allowed).
func NotAllowed(message string) *Error {
	return newError(http.StatusConflict, CodeNotAllowed, message)
}

// Forbidden refuses an action the person's roles do not allow, or a record outside their access, with a
// message for the user that says what to do (403 forbidden, feature 014 D-227). Nothing is changed.
func Forbidden(message string) *Error {
	return newError(http.StatusForbidden, CodeForbidden, message)
}

// NotReady refuses to publish or submit while needed fields are missing, listing every one of them grouped
// by record (409 not_ready).
func NotReady(message string, missing []ReadinessGroup) *Error {
	e := newError(http.StatusConflict, CodeNotReady, message)
	e.Missing = missing
	return e
}

// ParentDeletedPermanently refuses to restore a record of the kind level ("activity") whose parent, or a
// record above it, was deleted permanently: parent is that record and parentLevel its kind ("initiative").
// Nothing can bring it back, so the record cannot be restored (011 FR-024).
func ParentDeletedPermanently(level string, parent RecordRef, parentLevel string) *Error {
	message := fmt.Sprintf("This %s can't be restored because its %s '%s' was permanently deleted.",
		level, parentLevel, parent.Name)
	e := newError(http.StatusConflict, CodeParentDeletedPermanently, message)
	e.Parent = &parent
	return e
}

// Internal reports an unexpected failure without technical details (FR-020).
func Internal() *Error {
	return newError(http.StatusInternalServerError, CodeInternalError, internalMessage)
}

// Unavailable reports, with 503 and the code of Internal, that the API cannot serve requests now because
// its data cannot be used.
func Unavailable() *Error {
	return newError(http.StatusServiceUnavailable, CodeInternalError, internalMessage)
}

// Problems collects the field problems of one request, so they are all reported at once (FR-020):
//
//	var problems errs.Problems
//	if name == "" {
//		problems.Add("name", "Enter a name.")
//	}
//	if err := problems.Err(); err != nil {
//		return err
//	}
type Problems []FieldError

// Add records a problem with one field.
func (p *Problems) Add(field, message string) {
	*p = append(*p, FieldError{Field: field, Message: message})
}

// Err returns a validation error listing every problem, or nil when there is none.
func (p Problems) Err() error {
	if len(p) == 0 {
		return nil // a plain nil, so that "err != nil" is false
	}
	return Validation(p...)
}
