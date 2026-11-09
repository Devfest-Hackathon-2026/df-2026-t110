package usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

type fakeProvider struct {
	mu        sync.Mutex
	identity  usecase.Identity
	err       error
	completed int
}

func (p *fakeProvider) Begin() (usecase.AuthorizationRequest, error) {
	return usecase.AuthorizationRequest{URL: "https://accounts.example/authorize", State: "state", Nonce: "nonce", CodeVerifier: "verifier"}, nil
}

func (p *fakeProvider) Complete(_ context.Context, request usecase.AuthorizationRequest, state, code string) (usecase.Identity, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.completed++
	if p.err != nil {
		return usecase.Identity{}, p.err
	}
	if state != request.State || code != "code" {
		return usecase.Identity{}, errors.New("rejected")
	}
	return p.identity, nil
}

func (p *fakeProvider) completions() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.completed
}

type clock struct {
	mu  sync.Mutex
	now time.Time
}

type loginAttemptsStub struct {
	attempts map[string]usecase.LoginAttempt
}

func (s *loginAttemptsStub) Save(_ context.Context, attempt usecase.LoginAttempt) error {
	s.attempts[attempt.ID] = attempt
	return nil
}

func (s *loginAttemptsStub) Consume(_ context.Context, id string) (usecase.LoginAttempt, error) {
	attempt, ok := s.attempts[id]
	if !ok {
		return usecase.LoginAttempt{}, usecase.ErrNotFound
	}
	delete(s.attempts, id)
	return attempt, nil
}

type usersStub struct{ users map[string]domain.User }

func (s *usersStub) ResolveIdentity(_ context.Context, subject string) (domain.User, error) {
	if user, ok := s.users[subject]; ok {
		return user, nil
	}
	user := domain.User{GoogleSubject: subject}
	s.users[subject] = user
	return user, nil
}

func (s *usersStub) Get(_ context.Context, subject string) (domain.User, error) {
	user, ok := s.users[subject]
	if !ok {
		return domain.User{}, usecase.ErrNotFound
	}
	return user, nil
}

type sessionsStub struct{ sessions map[string]domain.Session }

func (s *sessionsStub) Save(_ context.Context, session domain.Session) error {
	s.sessions[session.ID] = session
	return nil
}

func (s *sessionsStub) Get(_ context.Context, id string) (domain.Session, error) {
	session, ok := s.sessions[id]
	if !ok {
		return domain.Session{}, usecase.ErrNotFound
	}
	return session, nil
}

func (s *sessionsStub) Delete(_ context.Context, id string) error {
	delete(s.sessions, id)
	return nil
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type authFixture struct {
	auth     *usecase.Auth
	provider *fakeProvider
	clock    *clock
	sessions *sessionsStub
	session  *usecase.Session
}

func newAuthFixture() *authFixture {
	c := &clock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	p := &fakeProvider{identity: usecase.Identity{Subject: "google-subject"}}
	sessions := &sessionsStub{sessions: map[string]domain.Session{}}
	users := &usersStub{users: map[string]domain.User{}}
	session := usecase.NewSession(usecase.SessionDeps{Sessions: sessions, Users: users, Now: c.Now, TTL: time.Hour})
	auth := usecase.NewAuth(usecase.AuthDeps{
		Provider:        p,
		LoginAttempts:   &loginAttemptsStub{attempts: map[string]usecase.LoginAttempt{}},
		Users:           users,
		Sessions:        session,
		Now:             c.Now,
		LoginAttemptTTL: time.Minute,
	})
	return &authFixture{auth: auth, provider: p, clock: c, sessions: sessions, session: session}
}

func (f *authFixture) login(t *testing.T) domain.Session {
	t.Helper()
	attempt, err := f.auth.BeginLogin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	session, err := f.auth.CompleteLogin(context.Background(), usecase.LoginCallback{AttemptID: attempt.ID, State: "state", Code: "code"})
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func TestBeginLoginStoresAttemptWithProviderURL(t *testing.T) {
	t.Parallel()
	f := newAuthFixture()

	attempt, err := f.auth.BeginLogin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if attempt.ID == "" {
		t.Error("attempt ID is empty")
	}
	if attempt.Request.URL != "https://accounts.example/authorize" {
		t.Errorf("attempt URL = %q", attempt.Request.URL)
	}
	if want := f.clock.Now().Add(time.Minute); !attempt.ExpiresAt.Equal(want) {
		t.Errorf("attempt expires at %v, want %v", attempt.ExpiresAt, want)
	}
	other, err := f.auth.BeginLogin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if other.ID == attempt.ID {
		t.Error("two attempts share an ID")
	}
}

func TestLoginWithoutProviderIsUnavailable(t *testing.T) {
	t.Parallel()
	auth := usecase.NewAuth(usecase.AuthDeps{})

	if _, err := auth.BeginLogin(context.Background()); !errors.Is(err, usecase.ErrLoginUnavailable) {
		t.Errorf("BeginLogin() error = %v, want ErrLoginUnavailable", err)
	}
	if _, err := auth.CompleteLogin(context.Background(), usecase.LoginCallback{AttemptID: "a"}); !errors.Is(err, usecase.ErrLoginUnavailable) {
		t.Errorf("CompleteLogin() error = %v, want ErrLoginUnavailable", err)
	}
}

func TestCompleteLoginStartsSessionForIdentifiedUser(t *testing.T) {
	t.Parallel()
	f := newAuthFixture()

	session := f.login(t)

	if session.ID == "" {
		t.Error("session ID is empty")
	}
	if want := f.clock.Now().Add(time.Hour); !session.ExpiresAt.Equal(want) {
		t.Errorf("session expires at %v, want %v", session.ExpiresAt, want)
	}
	user, err := f.session.CurrentUser(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if user.GoogleSubject != session.GoogleSubject || user.GoogleSubject != "google-subject" || user.Name != nil {
		t.Errorf("CurrentUser() = %+v, session user %q", user, session.GoogleSubject)
	}
}

func TestCompleteLoginReusesUserOfSameSubject(t *testing.T) {
	t.Parallel()
	f := newAuthFixture()
	first := f.login(t)

	second := f.login(t)

	if second.ID == first.ID {
		t.Error("second login reused the session ID")
	}
	if second.GoogleSubject != first.GoogleSubject {
		t.Errorf("user subject = %q, want %q", second.GoogleSubject, first.GoogleSubject)
	}
	user, err := f.session.CurrentUser(context.Background(), second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if user.Name != nil {
		t.Errorf("user name = %v, want no name", *user.Name)
	}
}

func TestCompleteLoginRejectsInvalidAttempts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		// arrange returns the callback to complete, after any setup.
		arrange        func(t *testing.T, f *authFixture, attemptID string) usecase.LoginCallback
		wantCompletion bool
	}{
		{
			name: "missing attempt ID",
			arrange: func(_ *testing.T, _ *authFixture, _ string) usecase.LoginCallback {
				return usecase.LoginCallback{State: "state", Code: "code"}
			},
		},
		{
			name: "unknown attempt ID",
			arrange: func(_ *testing.T, _ *authFixture, _ string) usecase.LoginCallback {
				return usecase.LoginCallback{AttemptID: "unknown", State: "state", Code: "code"}
			},
		},
		{
			name: "replayed attempt",
			arrange: func(t *testing.T, f *authFixture, attemptID string) usecase.LoginCallback {
				t.Helper()
				callback := usecase.LoginCallback{AttemptID: attemptID, State: "state", Code: "code"}
				if _, err := f.auth.CompleteLogin(context.Background(), callback); err != nil {
					t.Fatalf("first CompleteLogin() error = %v", err)
				}
				return callback
			},
			wantCompletion: true, // by the first, successful login only
		},
		{
			name: "attempt expired exactly at its TTL",
			arrange: func(_ *testing.T, f *authFixture, attemptID string) usecase.LoginCallback {
				f.clock.Advance(time.Minute)
				return usecase.LoginCallback{AttemptID: attemptID, State: "state", Code: "code"}
			},
		},
		{
			name: "provider returned an error",
			arrange: func(_ *testing.T, _ *authFixture, attemptID string) usecase.LoginCallback {
				return usecase.LoginCallback{AttemptID: attemptID, State: "state", ProviderError: "access_denied"}
			},
		},
		{
			name: "provider rejected the response",
			arrange: func(_ *testing.T, _ *authFixture, attemptID string) usecase.LoginCallback {
				return usecase.LoginCallback{AttemptID: attemptID, State: "forged", Code: "code"}
			},
			wantCompletion: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newAuthFixture()
			attempt, err := f.auth.BeginLogin(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			callback := tt.arrange(t, f, attempt.ID)

			_, err = f.auth.CompleteLogin(context.Background(), callback)

			if !errors.Is(err, usecase.ErrInvalidLogin) {
				t.Errorf("CompleteLogin() error = %v, want ErrInvalidLogin", err)
			}
			if got := f.provider.completions() > 0; got != tt.wantCompletion {
				t.Errorf("provider completed = %v, want %v", got, tt.wantCompletion)
			}
		})
	}
}

func TestFailedLoginConsumesAttempt(t *testing.T) {
	t.Parallel()
	f := newAuthFixture()
	attempt, err := f.auth.BeginLogin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.auth.CompleteLogin(context.Background(), usecase.LoginCallback{AttemptID: attempt.ID, State: "forged", Code: "code"}); err == nil {
		t.Fatal("CompleteLogin() with forged state succeeded")
	}

	_, err = f.auth.CompleteLogin(context.Background(), usecase.LoginCallback{AttemptID: attempt.ID, State: "state", Code: "code"})

	if !errors.Is(err, usecase.ErrInvalidLogin) {
		t.Errorf("retry error = %v, want ErrInvalidLogin", err)
	}
}
