package repo

import (
	"context"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo/gen/db"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ usecase.BattleProgramRepository = (*BattlePrograms)(nil)

type BattlePrograms struct{ queries *db.Queries }

func NewBattlePrograms(pool *pgxpool.Pool) *BattlePrograms {
	return &BattlePrograms{queries: db.New(pool)}
}

func (r *BattlePrograms) Set(ctx context.Context, subject string, ids []string) error {
	if len(ids) != 3 {
		return fmt.Errorf("expected three battle programs")
	}
	err := r.queries.SetBattlePrograms(ctx, db.SetBattleProgramsParams{
		GoogleSubject: subject, Program1: ids[0], Program2: ids[1], Program3: ids[2],
	})
	if err != nil {
		return fmt.Errorf("upsert battle programs: %w", err)
	}
	return nil
}
