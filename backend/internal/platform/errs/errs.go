// Package errs defines MAPP's one error format: {"error": {"code", "message", "fields", …}}, with messages written
// for users and codes for the web app's logic. Services return these errors and web.RespondError answers them.
package errs

import (
	"fmt"
	"net/http"
	"time"

	"mapp/internal/platform/actor"
)

const (
	CodeValidationFailed         = "validation_failed"
	CodeBadRequest               = "bad_request"
	CodeSignInRequired           = "sign_in_required"
	CodeSessionExpired           = "session_expired"
	CodeIncorrectCredentials     = "incorrect_credentials"
	CodeNotFound                 = "not_found"
	CodeVersionConflict          = "version_conflict"
	CodeRestoreParentFirst       = "restore_parent_first"
	CodeUndoExpired              = "undo_expired"
	CodeAlreadyResolved          = "already_resolved"
	CodeNameTaken                = "name_taken"
	CodeInternalError            = "internal_error"
	CodeInUse                    = "in_use"
	CodeCannotRestore            = "cannot_restore"
	CodeNotAllowed               = "not_allowed"
	CodeNotReady                 = "not_ready"
	CodeForbidden                = "forbidden"
	CodeParentDeletedPermanently = "parent_deleted_permanently"
	CodeTooManyAttempts          = "too_many_attempts"
	CodeTooLarge                 = "too_large"
)

// internalMessage hides what went wrong; the details go only to the server log.
const internalMessage = "Something went wrong on our side. Please try again."

// Error is an API failure; Status and RetryAfter are sent as the HTTP status and the Retry-After header.
type Error struct {
	Status     int              `json:"-"`
	RetryAfter time.Duration    `json:"-"`
	Code       string           `json:"code"`
	Message    string           `json:"message"`
	Fields     []FieldError     `json:"fields"` // never nil, so the JSON always has a list
	Conflict   *Conflict        `json:"conflict,omitempty"`
	Parent     *RecordRef       `json:"parent,omitempty"`
	Resolution *Resolution      `json:"resolution,omitempty"`
	Missing    []ReadinessGroup `json:"missing,omitempty"`
}

func (e *Error) Error() string {
	return e.Code + ": " + e.Message
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Conflict struct {
	ChangedBy string    `json:"changedBy"`
	ChangedAt time.Time `json:"changedAt"`
}

type RecordRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// MissingField is the one MissingField of the Plan module (D-193); settings.MissingField is this type.
type MissingField struct {
	Field  string `json:"field"`
	Label  string `json:"label"`
	Reason string `json:"reason,omitempty"`
}

type ReadinessGroup struct {
	Record RecordRef      `json:"record"`
	Fields []MissingField `json:"fields"`
}

type Resolution struct {
	How string      `json:"how"`
	By  actor.Actor `json:"by"`
	At  time.Time   `json:"at"`
}

func newError(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message, Fields: []FieldError{}}
}

func Validation(fields ...FieldError) *Error {
	e := newError(http.StatusBadRequest, CodeValidationFailed, "Some fields need your attention.")
	e.Fields = append(e.Fields, fields...)
	return e
}

func BadRequest(message string) *Error {
	return newError(http.StatusBadRequest, CodeBadRequest, message)
}

func SignInRequired() *Error {
	return newError(http.StatusUnauthorized, CodeSignInRequired, "Please sign in to continue.")
}

func SessionExpired() *Error {
	return newError(http.StatusUnauthorized, CodeSessionExpired, "Please sign in again")
}

// IncorrectCredentials never says whether the e-mail or the password was wrong.
func IncorrectCredentials() *Error {
	return newError(http.StatusUnauthorized, CodeIncorrectCredentials, "Incorrect e-mail or password.")
}

func NotFound(message string) *Error {
	return newError(http.StatusNotFound, CodeNotFound, message)
}

// VersionConflict's message has no time: only the browser knows the user's time zone (research R5).
func VersionConflict(noun, changedBy string, changedAt time.Time) *Error {
	message := fmt.Sprintf("Not saved — %s changed this %s", changedBy, noun)
	e := newError(http.StatusConflict, CodeVersionConflict, message)
	e.Conflict = &Conflict{ChangedBy: changedBy, ChangedAt: changedAt}
	return e
}

func RestoreParentFirst(parent RecordRef) *Error {
	message := fmt.Sprintf("Restore '%s' first", parent.Name)
	e := newError(http.StatusConflict, CodeRestoreParentFirst, message)
	e.Parent = &parent
	return e
}

func UndoExpired() *Error {
	return newError(http.StatusConflict, CodeUndoExpired,
		"Undo is no longer available — restore it from the recycle bin")
}

func AlreadyResolved(name string, r Resolution) *Error {
	message := fmt.Sprintf("'%s' is no longer in the recycle bin.", name)
	e := newError(http.StatusConflict, CodeAlreadyResolved, message)
	e.Resolution = &r
	return e
}

func NameTaken(message string) *Error {
	return newError(http.StatusConflict, CodeNameTaken, message)
}

func InUse(message string) *Error {
	return newError(http.StatusConflict, CodeInUse, message)
}

func CannotRestore(message string) *Error {
	return newError(http.StatusConflict, CodeCannotRestore, message)
}

func NotAllowed(message string) *Error {
	return newError(http.StatusConflict, CodeNotAllowed, message)
}

func Forbidden(message string) *Error {
	return newError(http.StatusForbidden, CodeForbidden, message)
}

func NotReady(message string, missing []ReadinessGroup) *Error {
	e := newError(http.StatusConflict, CodeNotReady, message)
	e.Missing = missing
	return e
}

func ParentDeletedPermanently(level string, parent RecordRef, parentLevel string) *Error {
	message := fmt.Sprintf("This %s can't be restored because its %s '%s' was permanently deleted.",
		level, parentLevel, parent.Name)
	e := newError(http.StatusConflict, CodeParentDeletedPermanently, message)
	e.Parent = &parent
	return e
}

func TooManyAttempts(message string, retryAfter time.Duration) *Error {
	e := newError(http.StatusTooManyRequests, CodeTooManyAttempts, message)
	e.RetryAfter = retryAfter
	return e
}

func TooLarge(message string) *Error {
	return newError(http.StatusRequestEntityTooLarge, CodeTooLarge, message)
}

func Internal() *Error {
	return newError(http.StatusInternalServerError, CodeInternalError, internalMessage)
}

func Unavailable() *Error {
	return newError(http.StatusServiceUnavailable, CodeInternalError, internalMessage)
}

type Problems []FieldError

func (p *Problems) Add(field, message string) {
	*p = append(*p, FieldError{Field: field, Message: message})
}

func (p Problems) Err() error {
	if len(p) == 0 {
		return nil // a plain nil, so that "err != nil" is false
	}
	return Validation(p...)
}
