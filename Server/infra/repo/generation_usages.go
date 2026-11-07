package repo

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo/gen/db"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

var _ usecase.GenerationQuotaRepository = (*GenerationUsages)(nil)

// GenerationUsages records generation token reservations in PostgreSQL.
type GenerationUsages struct {
	pool *pgxpool.Pool
}

func NewGenerationUsages(pool *pgxpool.Pool) *GenerationUsages {
	return &GenerationUsages{pool: pool}
}

// ReserveGenerationTokens serializes reservations by locking the user's row,
// which is released when the transaction ends before this method returns.
func (r *GenerationUsages) ReserveGenerationTokens(ctx context.Context, subject string, windowStart time.Time, tokens int, decide func(used int) error) error {
	if tokens <= 0 || tokens > math.MaxInt32 {
		return usecase.ErrInvalidQuotaReservation
	}
	reserved := int32(tokens)
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		queries := db.New(tx)
		if _, err := queries.LockUserForGeneration(ctx, subject); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return usecase.ErrNotFound
			}
			return fmt.Errorf("lock user: %w", err)
		}
		used, err := queries.SumGenerationTokensSince(ctx, db.SumGenerationTokensSinceParams{
			GoogleSubject: subject,
			WindowStart:   pgtype.Timestamptz{Time: windowStart, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("sum generation tokens: %w", err)
		}
		if err := decide(int(used)); err != nil {
			return err
		}
		if err := queries.CreateGenerationUsage(ctx, db.CreateGenerationUsageParams{
			GoogleSubject: subject,
			Tokens:        reserved,
		}); err != nil {
			return fmt.Errorf("record generation usage: %w", err)
		}
		return nil
	})
}
