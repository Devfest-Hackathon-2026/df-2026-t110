package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

func TestSessionsSaveAndGet(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	mustCreateUser(t, pool, "sub", nil)
	sessions := NewSessions(pool)
	ctx := context.Background()
	expiresAt := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)

	if err := sessions.Save(ctx, domain.Session{ID: "s1", GoogleSubject: "sub", ExpiresAt: expiresAt}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	// Saving the same ID again replaces the stored expiry.
	extended := expiresAt.Add(time.Hour)
	if err := sessions.Save(ctx, domain.Session{ID: "s1", GoogleSubject: "sub", ExpiresAt: extended}); err != nil {
		t.Fatalf("Save() again error = %v", err)
	}

	got, err := sessions.Get(ctx, "s1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != "s1" || got.GoogleSubject != "sub" {
		t.Errorf("Get() = %+v, want ID s1 and subject sub", got)
	}
	if !got.ExpiresAt.Equal(extended) {
		t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, extended)
	}
}

func TestSessionsGetMissing(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)

	_, err := NewSessions(pool).Get(context.Background(), "missing")
	if !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("Get() error = %v, want %v", err, usecase.ErrNotFound)
	}
}

func TestSessionsDelete(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	mustCreateUser(t, pool, "sub", nil)
	sessions := NewSessions(pool)
	ctx := context.Background()
	if err := sessions.Save(ctx, domain.Session{ID: "s1", GoogleSubject: "sub", ExpiresAt: time.Now()}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := sessions.Delete(ctx, "s1"); err != nil {
		t.Errorf("Delete() error = %v", err)
	}
	if _, err := sessions.Get(ctx, "s1"); !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("Get() after Delete() error = %v, want %v", err, usecase.ErrNotFound)
	}
	if err := sessions.Delete(ctx, "s1"); err != nil {
		t.Errorf("Delete() of a missing session error = %v, want nil", err)
	}
}
