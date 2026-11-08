package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LoginAttempts struct{ pool *pgxpool.Pool }

func NewLoginAttempts(pool *pgxpool.Pool) *LoginAttempts { return &LoginAttempts{pool: pool} }

func (r *LoginAttempts) Save(ctx context.Context, a usecase.LoginAttempt) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO login_attempts (id, state, nonce, code_verifier, expires_at) VALUES ($1, $2, $3, $4, $5)`, a.ID, a.Request.State, a.Request.Nonce, a.Request.CodeVerifier, a.ExpiresAt)
	if err != nil {
		return fmt.Errorf("save login attempt: %w", err)
	}
	return nil
}

func (r *LoginAttempts) Consume(ctx context.Context, id string) (usecase.LoginAttempt, error) {
	var a usecase.LoginAttempt
	a.ID = id
	err := r.pool.QueryRow(ctx, `DELETE FROM login_attempts WHERE id = $1 RETURNING state, nonce, code_verifier, expires_at`, id).Scan(&a.Request.State, &a.Request.Nonce, &a.Request.CodeVerifier, &a.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return usecase.LoginAttempt{}, usecase.ErrNotFound
	}
	if err != nil {
		return usecase.LoginAttempt{}, fmt.Errorf("consume login attempt: %w", err)
	}
	return a, nil
}
