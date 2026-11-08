package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

func TestLoginAttemptIsConsumedOnce(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	store := NewLoginAttempts(pool)
	ctx := context.Background()
	attempt := usecase.LoginAttempt{ID: "attempt", Request: usecase.AuthorizationRequest{State: "state", Nonce: "nonce", CodeVerifier: "verifier"}, ExpiresAt: time.Now().Add(time.Minute)}
	if err := store.Save(ctx, attempt); err != nil {
		t.Fatal(err)
	}
	got, err := store.Consume(ctx, attempt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Request.State != attempt.Request.State || got.Request.Nonce != attempt.Request.Nonce || got.Request.CodeVerifier != attempt.Request.CodeVerifier {
		t.Errorf("consumed attempt = %+v", got)
	}
	if _, err := store.Consume(ctx, attempt.ID); !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("second consume = %v, want ErrNotFound", err)
	}
}
