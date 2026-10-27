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
)

type Service struct {
	tx           Transactor
	repo         Store
	people       People
	demoMode     bool
	demoPassword string
	now          clock.Clock
}

type Options struct {
	DemoMode     bool
	DemoPassword string
	Now          clock.Clock
}

func NewService(tx Transactor, repo Store, peopleRepo People, opts Options) *Service {
	return &Service{
		tx: tx, repo: repo, people: peopleRepo, demoMode: opts.DemoMode, demoPassword: opts.DemoPassword,
		now: opts.Now,
	}
}

func (s *Service) DemoMode() bool {
	return s.demoMode
}

// DemoPassword is shown openly on the sign-in page: all demo data is fictitious.
func (s *Service) DemoPassword() string {
	return s.demoPassword
}

// SignIn ends oldToken's session and every expired one of anyone, so old sessions do not pile up.
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
		return "", people.Person{}, err
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

// checkCredentials returns one error whatever was wrong, so the answer never tells which e-mails exist.
func (s *Service) checkCredentials(ctx context.Context, email, password string) (people.Person, error) {
	person, found, err := s.people.FindByEmail(ctx, email)
	if err != nil {
		return people.Person{}, err
	}
	// Constant time: the time taken never hints at how much of the password was right.
	passwordRight := subtle.ConstantTimeCompare([]byte(password), []byte(s.demoPassword)) == 1
	if !found || !person.Active || !passwordRight {
		return people.Person{}, errs.IncorrectCredentials()
	}
	return person, nil
}

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

func (s *Service) SignOut(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.tx.Write(ctx, func(ctx context.Context) error {
		return s.repo.Delete(ctx, hashToken(token))
	})
}

// Authenticate deletes a refused session, all in one Write: a session signed out meanwhile never comes back.
func (s *Service) Authenticate(ctx context.Context, token string) (actor.Actor, error) {
	if token == "" {
		return actor.Actor{}, errs.SignInRequired()
	}

	var a actor.Actor
	// refusal is returned after the Write: an error from inside it would undo the deletion of the session.
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

// newToken writes 32 random bytes as base64url, so they fit in a cookie.
func newToken() string {
	random := make([]byte, 32)
	rand.Read(random) // never fails: Go stops the program when the system cannot give random bytes
	return base64.RawURLEncoding.EncodeToString(random)
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
