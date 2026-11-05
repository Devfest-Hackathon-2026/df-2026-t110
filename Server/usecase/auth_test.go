package usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo"
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
	sessions *repo.Sessions
}

func newAuthFixture() *authFixture {
	c := &clock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	p := &fakeProvider{identity: usecase.Identity{Subject: "google-subject", Name: "Alice"}}
	sessions := repo.NewSessions()
	auth := usecase.NewAuth(usecase.AuthDeps{
		Provider:        p,
		LoginAttempts:   repo.NewLoginAttempts(c.Now),
		Users:           repo.NewUsers(),
		Sessions:        sessions,
		Now:             c.Now,
		LoginAttemptTTL: time.Minute,
		SessionTTL:      time.Hour,
	})
	return &authFixture{auth: auth, provider: p, clock: c, sessions: sessions}
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
	auth := usecase.NewAuth(usecase.AuthDeps{LoginAttempts: repo.NewLoginAttempts(nil), Users: repo.NewUsers(), Sessions: repo.NewSessions()})

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
	user, err := f.auth.CurrentUser(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != session.UserID || user.GoogleSubject != "google-subject" || user.Name != "Alice" {
		t.Errorf("CurrentUser() = %+v, session user %q", user, session.UserID)
	}
}

func TestCompleteLoginReusesUserOfSameSubject(t *testing.T) {
	t.Parallel()
	f := newAuthFixture()
	first := f.login(t)
	f.provider.identity.Name = "Renamed"

	second := f.login(t)

	if second.ID == first.ID {
		t.Error("second login reused the session ID")
	}
	if second.UserID != first.UserID {
		t.Errorf("user ID = %q, want %q", second.UserID, first.UserID)
	}
	user, err := f.auth.CurrentUser(context.Background(), second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if user.Name != "Alice" {
		t.Errorf("user name = %q, want the name stored at first login", user.Name)
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

func TestCurrentUser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		arrange func(t *testing.T, f *authFixture, sessionID string) string
		wantErr error
	}{
		{name: "live session", arrange: func(_ *testing.T, _ *authFixture, id string) string { return id }},
		{name: "just before expiry", arrange: func(_ *testing.T, f *authFixture, id string) string {
			f.clock.Advance(time.Hour - time.Nanosecond)
			return id
		}},
		{name: "expired exactly at its TTL", arrange: func(_ *testing.T, f *authFixture, id string) string { f.clock.Advance(time.Hour); return id }, wantErr: usecase.ErrUnauthenticated},
		{name: "no session cookie", arrange: func(_ *testing.T, _ *authFixture, _ string) string { return "" }, wantErr: usecase.ErrUnauthenticated},
		{name: "unknown session", arrange: func(_ *testing.T, _ *authFixture, _ string) string { return "unknown" }, wantErr: usecase.ErrUnauthenticated},
		{
			name: "logged out",
			arrange: func(t *testing.T, f *authFixture, id string) string {
				t.Helper()
				if err := f.auth.Logout(context.Background(), id); err != nil {
					t.Fatalf("Logout() error = %v", err)
				}
				return id
			},
			wantErr: usecase.ErrUnauthenticated,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newAuthFixture()
			session := f.login(t)

			_, err := f.auth.CurrentUser(context.Background(), tt.arrange(t, f, session.ID))

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("CurrentUser() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCurrentUserDeletesExpiredSession(t *testing.T) {
	t.Parallel()
	f := newAuthFixture()
	session := f.login(t)
	f.clock.Advance(time.Hour)

	if _, err := f.auth.CurrentUser(context.Background(), session.ID); !errors.Is(err, usecase.ErrUnauthenticated) {
		t.Errorf("CurrentUser() error = %v, want ErrUnauthenticated", err)
	}
	if _, err := f.sessions.Get(context.Background(), session.ID); !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("expired session still stored: err = %v", err)
	}
}

func TestLogoutWithoutSessionSucceeds(t *testing.T) {
	t.Parallel()
	f := newAuthFixture()

	for _, id := range []string{"", "unknown"} {
		if err := f.auth.Logout(context.Background(), id); err != nil {
			t.Errorf("Logout(%q) error = %v", id, err)
		}
	}
}
