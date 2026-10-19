// Package auth is NorthStar's demo sign-in (decision D-004, research R6): a person signs in with a demo
// e-mail and the shared demo password, gets a session kept in an HttpOnly cookie, and every request that
// needs a signed-in person is turned into an actor by RequireSignIn. Microsoft Entra ID will later replace
// the sign-in, and only the middleware that produces the actor (ADR-0005).
package auth

import (
	"time"

	"northstar/internal/people"
)

// CookieName is the name of the cookie that carries the session token to and from the browser.
const CookieName = "northstar_session"

const (
	// idleLimit is how long a session lasts without a request (FR-006): 12 hours, plus the minute by which
	// lastUsedAt can lag behind the last request, so that no session ends before 12 full hours.
	idleLimit = 12*time.Hour + time.Minute

	// touchInterval is how old lastUsedAt must be before a request rewrites it, so that most requests save
	// nothing.
	touchInterval = time.Minute
)

// Session is a person signed in in one browser (data model §3), stored in the table "sessions". Its id is
// the hash of its token: the token itself is never stored, so a copied data file cannot be used to sign
// in.
type Session struct {
	ID         string    `json:"id"`         // the hex SHA-256 of the session token
	PersonID   string    `json:"personId"`   // the signed-in person
	CreatedAt  time.Time `json:"createdAt"`  // when the person signed in
	LastUsedAt time.Time `json:"lastUsedAt"` // the last request, rewritten when it is at least a minute old
}

// expiredAt reports whether the session has ended by now: more than idleLimit after its last request.
func (s Session) expiredAt(now time.Time) bool {
	return now.Sub(s.LastUsedAt) > idleLimit
}

// DemoPeople is the answer of GET /api/auth/demo-people: what the sign-in page shows (FR-001, FR-002).
type DemoPeople struct {
	Password string              `json:"password"` // the shared demo password, shown openly (D-004)
	People   []people.DemoPerson `json:"people"`   // the active demo people, sorted by name; empty without demo data
}
