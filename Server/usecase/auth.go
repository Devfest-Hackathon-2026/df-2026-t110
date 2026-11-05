package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

const (
	DefaultLoginAttemptTTL = 10 * time.Minute
	DefaultSessionTTL      = 7 * 24 * time.Hour
)

// AuthorizationRequest is one sign-in attempt issued by an IdentityProvider.
// Only URL may be sent to the browser; the rest must stay server-side.
type AuthorizationRequest struct {
	URL          string
	State        string
	Nonce        string
	CodeVerifier string
}

// Identity is a user identity verified by an IdentityProvider.
type Identity struct {
	Subject string
	Name    string
}

// IdentityProvider runs an OpenID Connect authorization code flow.
type IdentityProvider interface {
	Begin() (AuthorizationRequest, error)
	// Complete verifies the provider's response for request. Callers must have
	// consumed request from storage beforehand so that it cannot be replayed.
	Complete(ctx context.Context, request AuthorizationRequest, state, code string) (Identity, error)
}

// LoginAttempt is a stored AuthorizationRequest that the browser may complete once.
type LoginAttempt struct {
	ID        string
	Request   AuthorizationRequest
	ExpiresAt time.Time
}

// LoginAttemptRepository stores login attempts so that each can be consumed at most once.
type LoginAttemptRepository interface {
	Save(ctx context.Context, attempt LoginAttempt) error
	// Consume removes and returns the attempt, or returns ErrNotFound.
	Consume(ctx context.Context, id string) (LoginAttempt, error)
}

// UserRepository stores users.
type UserRepository interface {
	// CreateIfAbsentByGoogleSubject stores candidate unless a user with the same
	// GoogleSubject exists, and returns the stored user either way.
	CreateIfAbsentByGoogleSubject(ctx context.Context, candidate domain.User) (domain.User, error)
	// Get returns the user or ErrNotFound.
	Get(ctx context.Context, id string) (domain.User, error)
}

// SessionRepository stores sessions.
type SessionRepository interface {
	Save(ctx context.Context, session domain.Session) error
	// Get returns the session or ErrNotFound.
	Get(ctx context.Context, id string) (domain.Session, error)
	// Delete removes the session; deleting a missing session is not an error.
	Delete(ctx context.Context, id string) error
}

// AuthDeps are the dependencies of Auth. Now, LoginAttemptTTL, and SessionTTL
// fall back to time.Now and the package defaults when zero.
type AuthDeps struct {
	Provider        IdentityProvider
	LoginAttempts   LoginAttemptRepository
	Users           UserRepository
	Sessions        SessionRepository
	Now             func() time.Time
	LoginAttemptTTL time.Duration
	SessionTTL      time.Duration
}

// Auth signs users in with an external identity provider and manages sessions.
type Auth struct {
	deps AuthDeps
}

func NewAuth(deps AuthDeps) *Auth {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.LoginAttemptTTL <= 0 {
		deps.LoginAttemptTTL = DefaultLoginAttemptTTL
	}
	if deps.SessionTTL <= 0 {
		deps.SessionTTL = DefaultSessionTTL
	}
	return &Auth{deps: deps}
}

// BeginLogin stores a new login attempt and returns it; the caller binds
// attempt.ID to the browser and redirects to attempt.Request.URL.
func (a *Auth) BeginLogin(ctx context.Context) (LoginAttempt, error) {
	if a.deps.Provider == nil {
		return LoginAttempt{}, ErrLoginUnavailable
	}
	request, err := a.deps.Provider.Begin()
	if err != nil {
		return LoginAttempt{}, fmt.Errorf("begin login: %w", err)
	}
	id, err := newRandomID()
	if err != nil {
		return LoginAttempt{}, fmt.Errorf("generate login attempt ID: %w", err)
	}
	attempt := LoginAttempt{ID: id, Request: request, ExpiresAt: a.deps.Now().Add(a.deps.LoginAttemptTTL)}
	if err := a.deps.LoginAttempts.Save(ctx, attempt); err != nil {
		return LoginAttempt{}, fmt.Errorf("save login attempt: %w", err)
	}
	return attempt, nil
}

// LoginCallback carries the identity provider's redirect parameters.
type LoginCallback struct {
	AttemptID     string
	State         string
	Code          string
	ProviderError string
}

// CompleteLogin consumes the login attempt, verifies the provider's response,
// and starts a session for the identified user. Any rejection of the attempt
// itself is reported as ErrInvalidLogin.
func (a *Auth) CompleteLogin(ctx context.Context, callback LoginCallback) (domain.Session, error) {
	if a.deps.Provider == nil {
		return domain.Session{}, ErrLoginUnavailable
	}
	if callback.AttemptID == "" {
		return domain.Session{}, fmt.Errorf("%w: no login attempt", ErrInvalidLogin)
	}
	// Consume before anything else so that a failed or denied attempt cannot be retried.
	attempt, err := a.deps.LoginAttempts.Consume(ctx, callback.AttemptID)
	if errors.Is(err, ErrNotFound) {
		return domain.Session{}, fmt.Errorf("%w: unknown or used login attempt", ErrInvalidLogin)
	}
	if err != nil {
		return domain.Session{}, fmt.Errorf("consume login attempt: %w", err)
	}
	if !a.deps.Now().Before(attempt.ExpiresAt) {
		return domain.Session{}, fmt.Errorf("%w: login attempt expired", ErrInvalidLogin)
	}
	if callback.ProviderError != "" {
		return domain.Session{}, fmt.Errorf("%w: provider returned %q", ErrInvalidLogin, callback.ProviderError)
	}
	identity, err := a.deps.Provider.Complete(ctx, attempt.Request, callback.State, callback.Code)
	if err != nil {
		return domain.Session{}, fmt.Errorf("%w: %w", ErrInvalidLogin, err)
	}

	candidateID, err := newRandomID()
	if err != nil {
		return domain.Session{}, fmt.Errorf("generate user ID: %w", err)
	}
	user, err := a.deps.Users.CreateIfAbsentByGoogleSubject(ctx, domain.User{ID: candidateID, GoogleSubject: identity.Subject, Name: identity.Name})
	if err != nil {
		return domain.Session{}, fmt.Errorf("save user: %w", err)
	}
	sessionID, err := newRandomID()
	if err != nil {
		return domain.Session{}, fmt.Errorf("generate session ID: %w", err)
	}
	session := domain.Session{ID: sessionID, UserID: user.ID, ExpiresAt: a.deps.Now().Add(a.deps.SessionTTL)}
	if err := a.deps.Sessions.Save(ctx, session); err != nil {
		return domain.Session{}, fmt.Errorf("save session: %w", err)
	}
	return session, nil
}

// Logout ends the session. It succeeds even when the session does not exist.
func (a *Auth) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	if err := a.deps.Sessions.Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// CurrentUser returns the user of a live session, or ErrUnauthenticated.
func (a *Auth) CurrentUser(ctx context.Context, sessionID string) (domain.User, error) {
	if sessionID == "" {
		return domain.User{}, ErrUnauthenticated
	}
	session, err := a.deps.Sessions.Get(ctx, sessionID)
	if errors.Is(err, ErrNotFound) {
		return domain.User{}, ErrUnauthenticated
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get session: %w", err)
	}
	if session.Expired(a.deps.Now()) {
		if err := a.deps.Sessions.Delete(ctx, sessionID); err != nil {
			return domain.User{}, fmt.Errorf("delete expired session: %w", err)
		}
		return domain.User{}, ErrUnauthenticated
	}
	user, err := a.deps.Users.Get(ctx, session.UserID)
	if errors.Is(err, ErrNotFound) {
		return domain.User{}, ErrUnauthenticated
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}
