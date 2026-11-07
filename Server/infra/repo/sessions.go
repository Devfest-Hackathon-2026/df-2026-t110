package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo/gen/db"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

var _ usecase.SessionRepository = (*Sessions)(nil)

// Sessions stores sessions in PostgreSQL. Expired sessions are kept until
// deleted; callers check expiry with domain.Session.Expired.
type Sessions struct {
	queries *db.Queries
}

func NewSessions(pool *pgxpool.Pool) *Sessions {
	return &Sessions{queries: db.New(pool)}
}

func (r *Sessions) Save(ctx context.Context, session domain.Session) error {
	if err := r.queries.SaveSession(ctx, db.SaveSessionParams{
		ID:            session.ID,
		GoogleSubject: session.GoogleSubject,
		ExpiresAt:     pgtype.Timestamptz{Time: session.ExpiresAt, Valid: true},
	}); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return nil
}

func (r *Sessions) Get(ctx context.Context, id string) (domain.Session, error) {
	session, err := r.queries.GetSession(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Session{}, usecase.ErrNotFound
	}
	if err != nil {
		return domain.Session{}, fmt.Errorf("get session: %w", err)
	}
	return domain.Session{
		ID:            session.ID,
		GoogleSubject: session.GoogleSubject,
		ExpiresAt:     session.ExpiresAt.Time,
	}, nil
}

func (r *Sessions) Delete(ctx context.Context, id string) error {
	if err := r.queries.DeleteSession(ctx, id); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
