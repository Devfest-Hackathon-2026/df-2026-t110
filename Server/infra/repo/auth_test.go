package repo

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

func TestLoginAttemptsConsumeOnlyOnce(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	r := NewLoginAttempts(func() time.Time { return now })
	attempt := usecase.LoginAttempt{ID: "a", Request: usecase.AuthorizationRequest{State: "s"}, ExpiresAt: now.Add(time.Minute)}
	if err := r.Save(context.Background(), attempt); err != nil {
		t.Fatal(err)
	}

	const consumers = 16
	var wins atomic.Int32
	var wg sync.WaitGroup
	for range consumers {
		wg.Go(func() {
			got, err := r.Consume(context.Background(), "a")
			switch {
			case err == nil:
				wins.Add(1)
				if got != attempt {
					t.Errorf("Consume() = %+v, want %+v", got, attempt)
				}
			case !errors.Is(err, usecase.ErrNotFound):
				t.Errorf("Consume() error = %v, want ErrNotFound", err)
			}
		})
	}
	wg.Wait()

	if wins.Load() != 1 {
		t.Errorf("%d consumers succeeded, want 1", wins.Load())
	}
}

func TestLoginAttemptsSavePrunesExpired(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	r := NewLoginAttempts(func() time.Time { return now })
	for _, a := range []usecase.LoginAttempt{
		{ID: "expired", ExpiresAt: now},
		{ID: "live", ExpiresAt: now.Add(time.Nanosecond)},
		{ID: "new", ExpiresAt: now.Add(time.Minute)},
	} {
		if err := r.Save(context.Background(), a); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := r.Consume(context.Background(), "expired"); !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("expired attempt: error = %v, want ErrNotFound", err)
	}
	for _, id := range []string{"live", "new"} {
		if _, err := r.Consume(context.Background(), id); err != nil {
			t.Errorf("attempt %q: error = %v", id, err)
		}
	}
}

func TestUsersCreateIfAbsentByGoogleSubject(t *testing.T) {
	t.Parallel()
	r := NewUsers()

	const callers = 16
	ids := make([]string, callers)
	var wg sync.WaitGroup
	for i := range callers {
		wg.Go(func() {
			u, err := r.CreateIfAbsentByGoogleSubject(context.Background(), domain.User{ID: strconv.Itoa(i), GoogleSubject: "sub", Name: "Alice"})
			if err != nil {
				t.Errorf("CreateIfAbsentByGoogleSubject() error = %v", err)
			}
			ids[i] = u.ID
		})
	}
	wg.Wait()

	for _, id := range ids[1:] {
		if id != ids[0] {
			t.Fatalf("same subject got different user IDs: %v", ids)
		}
	}
	if u, err := r.Get(context.Background(), ids[0]); err != nil || u.GoogleSubject != "sub" {
		t.Errorf("Get(%q) = %+v, %v", ids[0], u, err)
	}
	other, err := r.CreateIfAbsentByGoogleSubject(context.Background(), domain.User{ID: "other", GoogleSubject: "other-sub"})
	if err != nil || other.ID != "other" {
		t.Errorf("new subject: got %+v, %v", other, err)
	}
	if _, err := r.Get(context.Background(), "missing"); !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("Get(missing) error = %v, want ErrNotFound", err)
	}
}

func TestSessions(t *testing.T) {
	t.Parallel()
	r := NewSessions()
	s := domain.Session{ID: "s", UserID: "u"}
	if err := r.Save(context.Background(), s); err != nil {
		t.Fatal(err)
	}

	if got, err := r.Get(context.Background(), "s"); err != nil || got != s {
		t.Errorf("Get() = %+v, %v; want %+v", got, err, s)
	}
	if err := r.Delete(context.Background(), "s"); err != nil {
		t.Errorf("Delete() error = %v", err)
	}
	if _, err := r.Get(context.Background(), "s"); !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("Get() after Delete error = %v, want ErrNotFound", err)
	}
	if err := r.Delete(context.Background(), "s"); err != nil {
		t.Errorf("Delete() of missing session error = %v", err)
	}
}
