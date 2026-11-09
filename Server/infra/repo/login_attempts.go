package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo/gen/db"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

var _ usecase.LoginAttemptRepository = (*LoginAttempts)(nil)

// LoginAttempts stores OIDC login attempts in PostgreSQL so that any server
// instance can complete a sign-in started on another one. The authorization
// URL is not stored because it is only needed when the attempt begins.
type LoginAttempts struct {
	queries *db.Queries
}

func NewLoginAttempts(pool *pgxpool.Pool) *LoginAttempts {
	return &LoginAttempts{queries: db.New(pool)}
}

// Save also sweeps expired attempts; abandoned sign-ins are otherwise never consumed.
func (r *LoginAttempts) Save(ctx context.Context, attempt usecase.LoginAttempt) error {
	if err := r.queries.DeleteExpiredLoginAttempts(ctx); err != nil {
		return fmt.Errorf("delete expired login attempts: %w", err)
	}
	if err := r.queries.SaveLoginAttempt(ctx, db.SaveLoginAttemptParams{
		ID:           attempt.ID,
		State:        attempt.Request.State,
		Nonce:        attempt.Request.Nonce,
		CodeVerifier: attempt.Request.CodeVerifier,
		ExpiresAt:    pgtype.Timestamptz{Time: attempt.ExpiresAt, Valid: true},
	}); err != nil {
		return fmt.Errorf("save login attempt: %w", err)
	}
	return nil
}

func (r *LoginAttempts) Consume(ctx context.Context, id string) (usecase.LoginAttempt, error) {
	row, err := r.queries.ConsumeLoginAttempt(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return usecase.LoginAttempt{}, usecase.ErrNotFound
	}
	if err != nil {
		return usecase.LoginAttempt{}, fmt.Errorf("consume login attempt: %w", err)
	}
	return usecase.LoginAttempt{
		ID: row.ID,
		Request: usecase.AuthorizationRequest{
			State:        row.State,
			Nonce:        row.Nonce,
			CodeVerifier: row.CodeVerifier,
		},
		ExpiresAt: row.ExpiresAt.Time,
	}, nil
}
