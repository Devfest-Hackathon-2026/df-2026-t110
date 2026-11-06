package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

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
				if err := f.session.Logout(context.Background(), id); err != nil {
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

			_, err := f.session.CurrentUser(context.Background(), tt.arrange(t, f, session.ID))

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

	if _, err := f.session.CurrentUser(context.Background(), session.ID); !errors.Is(err, usecase.ErrUnauthenticated) {
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
		if err := f.session.Logout(context.Background(), id); err != nil {
			t.Errorf("Logout(%q) error = %v", id, err)
		}
	}
}
