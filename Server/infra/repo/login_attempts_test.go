package repo

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

func TestLoginAttemptsConsumeOnce(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	attempts := NewLoginAttempts(newTestPool(t))
	saved := usecase.LoginAttempt{
		ID:        "attempt",
		Request:   usecase.AuthorizationRequest{URL: "https://accounts.example/authorize", State: "state", Nonce: "nonce", CodeVerifier: "verifier"},
		ExpiresAt: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := attempts.Save(ctx, saved); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	type result struct {
		attempt usecase.LoginAttempt
		err     error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			got, err := attempts.Consume(ctx, saved.ID)
			results <- result{got, err}
		})
	}
	wg.Wait()
	close(results)

	successes, missing := 0, 0
	for r := range results {
		switch {
		case r.err == nil:
			successes++
			want := saved
			want.Request.URL = "" // only sent to the browser, never stored
			if r.attempt.ID != want.ID || r.attempt.Request != want.Request || !r.attempt.ExpiresAt.Equal(want.ExpiresAt) {
				t.Errorf("Consume() = %+v, want %+v", r.attempt, want)
			}
		case errors.Is(r.err, usecase.ErrNotFound):
			missing++
		default:
			t.Errorf("Consume() error = %v", r.err)
		}
	}
	if successes != 1 || missing != 1 {
		t.Errorf("successes = %d, missing = %d, want 1 and 1", successes, missing)
	}
}

func TestLoginAttemptsSaveSweepsExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	attempts := NewLoginAttempts(newTestPool(t))
	if err := attempts.Save(ctx, usecase.LoginAttempt{ID: "expired", ExpiresAt: time.Now().Add(-time.Hour)}); err != nil {
		t.Fatalf("Save(expired) error = %v", err)
	}
	if err := attempts.Save(ctx, usecase.LoginAttempt{ID: "current", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatalf("Save(current) error = %v", err)
	}

	if _, err := attempts.Consume(ctx, "expired"); !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("Consume(expired) error = %v, want ErrNotFound", err)
	}
	if _, err := attempts.Consume(ctx, "current"); err != nil {
		t.Errorf("Consume(current) error = %v", err)
	}
}
