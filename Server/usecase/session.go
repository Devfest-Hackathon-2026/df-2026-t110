package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

const DefaultSessionTTL = 7 * 24 * time.Hour

type SessionRepository interface {
	Save(ctx context.Context, session domain.Session) error
	Get(ctx context.Context, id string) (domain.Session, error)
	Delete(ctx context.Context, id string) error
}

type SessionUsers interface {
	Get(ctx context.Context, subject string) (domain.User, error)
}

type SessionDeps struct {
	Sessions SessionRepository
	Users    SessionUsers
	Now      func() time.Time
	TTL      time.Duration
}

type Session struct{ deps SessionDeps }

func NewSession(deps SessionDeps) *Session {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.TTL <= 0 {
		deps.TTL = DefaultSessionTTL
	}
	return &Session{deps: deps}
}

// Start creates a session for an authenticated subject.
func (s *Session) Start(ctx context.Context, subject string) (domain.Session, error) {
	id, err := newRandomID()
	if err != nil {
		return domain.Session{}, fmt.Errorf("generate session ID: %w", err)
	}
	session := domain.Session{ID: id, GoogleSubject: subject, ExpiresAt: s.deps.Now().Add(s.deps.TTL)}
	if err := s.deps.Sessions.Save(ctx, session); err != nil {
		return domain.Session{}, fmt.Errorf("save session: %w", err)
	}
	return session, nil
}

// Logout ends a session, including an already absent one.
func (s *Session) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	if err := s.deps.Sessions.Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// CurrentUser returns the user of a live session, or ErrUnauthenticated.
func (s *Session) CurrentUser(ctx context.Context, sessionID string) (domain.User, error) {
	if sessionID == "" {
		return domain.User{}, ErrUnauthenticated
	}
	session, err := s.deps.Sessions.Get(ctx, sessionID)
	if errors.Is(err, ErrNotFound) {
		return domain.User{}, ErrUnauthenticated
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get session: %w", err)
	}
	if session.Expired(s.deps.Now()) {
		if err := s.deps.Sessions.Delete(ctx, sessionID); err != nil {
			return domain.User{}, fmt.Errorf("delete expired session: %w", err)
		}
		return domain.User{}, ErrUnauthenticated
	}
	user, err := s.deps.Users.Get(ctx, session.GoogleSubject)
	if errors.Is(err, ErrNotFound) {
		return domain.User{}, ErrUnauthenticated
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}
