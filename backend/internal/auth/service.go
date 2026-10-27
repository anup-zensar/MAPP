package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/store"
)

// Service holds the rules of demo sign-in and sessions (research R6).
type Service struct {
	tx           store.Transactor
	repo         Store
	people       People
	demoMode     bool
	demoPassword string
	now          clock.Clock
}

// NewService returns the sign-in service. It keeps sessions in repo, finds people in peopleRepo and works
// in transactions of tx. demoMode says whether demo sign-in is on, which it is only in the local and
// development environments (FR-009); demoPassword is the password all demo people share; now is the clock.
func NewService(tx store.Transactor, repo Store, peopleRepo People, demoMode bool, demoPassword string, now clock.Clock) *Service {
	return &Service{tx: tx, repo: repo, people: peopleRepo, demoMode: demoMode, demoPassword: demoPassword, now: now}
}

// DemoMode reports whether demo sign-in is on (FR-009).
func (s *Service) DemoMode() bool {
	return s.demoMode
}

// DemoPassword returns the password all demo people share. The sign-in page shows it openly, because all
// demo data is fictitious (FR-002, D-004).
func (s *Service) DemoPassword() string {
	return s.demoPassword
}

// SignIn signs a demo person in (FR-003) and returns the token of the new session, for the browser's
// cookie, and the person. The e-mail must belong to an active person, ignoring letter case and surrounding
// spaces, and the password must be the demo password exactly. It takes no actor: it is what creates one.
//
// oldToken is the token of the session this browser had, or "": that session ends, and so does every
// expired session of anyone, so old sessions do not pile up.
func (s *Service) SignIn(ctx context.Context, email, password, oldToken string) (string, people.Person, error) {
	email = strings.ToLower(strings.TrimSpace(email)) // e-mails are stored in lower case
	var problems errs.Problems
	if email == "" {
		problems.Add("email", "Enter your e-mail")
	}
	if password == "" {
		problems.Add("password", "Enter the password")
	}
	if err := problems.Err(); err != nil {
		return "", people.Person{}, err // every empty field at once (FR-004)
	}

	token := newToken()
	var person people.Person
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		var err error
		person, err = s.checkCredentials(ctx, email, password)
		if err != nil {
			return err
		}
		now := s.now()
		if err := s.removeOldSessions(ctx, oldToken, now); err != nil {
			return err
		}
		return s.repo.Put(ctx, Session{ID: hashToken(token), PersonID: person.ID, CreatedAt: now, LastUsedAt: now})
	})
	if err != nil {
		return "", people.Person{}, err
	}
	return token, person, nil
}

// checkCredentials returns the active person with email when password is the demo password. Otherwise it
// returns the same error whatever was wrong, so the answer never tells which e-mails exist (FR-004).
func (s *Service) checkCredentials(ctx context.Context, email, password string) (people.Person, error) {
	person, found, err := s.people.FindByEmail(ctx, email)
	if err != nil {
		return people.Person{}, err
	}
	// Compared in constant time, as passwords always should be: the time taken never hints at how much of
	// the password was right.
	passwordRight := subtle.ConstantTimeCompare([]byte(password), []byte(s.demoPassword)) == 1
	if !found || !person.Active || !passwordRight {
		return people.Person{}, errs.IncorrectCredentials()
	}
	return person, nil
}

// removeOldSessions deletes the session of oldToken, when there is one, and every session that has expired
// by now.
func (s *Service) removeOldSessions(ctx context.Context, oldToken string, now time.Time) error {
	if oldToken != "" {
		if err := s.repo.Delete(ctx, hashToken(oldToken)); err != nil {
			return err
		}
	}
	sessions, err := s.repo.List(ctx)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		if session.expiredAt(now) {
			if err := s.repo.Delete(ctx, session.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// SignOut ends the session of token (FR-006). It does nothing when token is "" or its session no longer
// exists.
func (s *Service) SignOut(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.tx.Write(ctx, func(ctx context.Context) error {
		return s.repo.Delete(ctx, hashToken(token))
	})
}

// Authenticate returns the person signed in with token, as the actor of the request (FR-007). It refuses
// with sign_in_required when there is no such session or its person no longer exists or is inactive, and
// with session_expired after more than 12 hours and 1 minute without a request (FR-006); a refused session
// is deleted. A session in use gets its lastUsedAt moved forward, at most once a minute.
//
// All of it happens in one Write, so a session that is signed out at the same moment is never brought back
// by the update of lastUsedAt.
func (s *Service) Authenticate(ctx context.Context, token string) (actor.Actor, error) {
	if token == "" {
		return actor.Actor{}, errs.SignInRequired() // no cookie: nothing to look up
	}

	var a actor.Actor
	// refusal says why the session cannot be used. It is returned after the Write, not from inside it,
	// because an error returned from inside would undo the deletion of the refused session.
	var refusal error
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		session, found, err := s.repo.Get(ctx, hashToken(token))
		if err != nil {
			return err
		}
		if !found {
			refusal = errs.SignInRequired()
			return nil
		}

		now := s.now()
		if session.expiredAt(now) {
			refusal = errs.SessionExpired()
			return s.repo.Delete(ctx, session.ID)
		}

		person, found, err := s.people.Get(ctx, session.PersonID)
		if err != nil {
			return err
		}
		if !found || !person.Active {
			refusal = errs.SignInRequired()
			return s.repo.Delete(ctx, session.ID)
		}

		if now.Sub(session.LastUsedAt) >= touchInterval {
			session.LastUsedAt = now
			if err := s.repo.Put(ctx, session); err != nil {
				return err
			}
		}
		a = person.Actor()
		return nil
	})
	if err != nil {
		return actor.Actor{}, err
	}
	if refusal != nil {
		return actor.Actor{}, refusal
	}
	return a, nil
}

// newToken returns a new session token: 32 random bytes, written as base64url so they fit in a cookie.
func newToken() string {
	random := make([]byte, 32)
	rand.Read(random) // never fails: Go stops the program when the system cannot give random bytes
	return base64.RawURLEncoding.EncodeToString(random)
}

// hashToken returns the id under which the session of token is stored: the hex SHA-256 of the token.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
