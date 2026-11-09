package repo

import (
	"context"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo/gen/db"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ usecase.BattleProgramRepository = (*BattlePrograms)(nil)

type BattlePrograms struct{ pool *pgxpool.Pool }

func NewBattlePrograms(pool *pgxpool.Pool) *BattlePrograms {
	return &BattlePrograms{pool: pool}
}

// Set replaces all slots atomically. The use case determines the required count.
func (r *BattlePrograms) Set(ctx context.Context, subject string, ids []string) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		queries := db.New(tx)
		// Serialize replacements even when the user has no existing slots.
		if _, err := queries.LockBattleProgramOwner(ctx, subject); err != nil {
			return fmt.Errorf("lock battle program owner: %w", err)
		}
		if err := queries.DeleteBattlePrograms(ctx, subject); err != nil {
			return fmt.Errorf("delete battle programs: %w", err)
		}
		if err := queries.InsertBattlePrograms(ctx, db.InsertBattleProgramsParams{GoogleSubject: subject, ProgramIds: ids}); err != nil {
			return fmt.Errorf("insert battle programs: %w", err)
		}
		return nil
	})
}

func (r *BattlePrograms) Get(ctx context.Context, subject string) ([]string, error) {
	ids, err := db.New(r.pool).GetBattlePrograms(ctx, subject)
	if err != nil {
		return nil, fmt.Errorf("select battle programs: %w", err)
	}
	if len(ids) == 0 {
		return nil, usecase.ErrNotFound
	}
	return ids, nil
}
