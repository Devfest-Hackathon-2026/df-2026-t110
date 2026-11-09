package repo

import (
	"context"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ usecase.BattleProgramRepository = (*BattlePrograms)(nil)

type BattlePrograms struct{ pool *pgxpool.Pool }

func NewBattlePrograms(pool *pgxpool.Pool) *BattlePrograms { return &BattlePrograms{pool: pool} }

func (r *BattlePrograms) Set(ctx context.Context, subject string, ids []string) error {
	if len(ids) != 3 {
		return fmt.Errorf("expected three battle programs")
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO battle_programs (google_subject, program_1, program_2, program_3)
 VALUES ($1, $2, $3, $4)
 ON CONFLICT (google_subject) DO UPDATE SET
 program_1 = EXCLUDED.program_1, program_2 = EXCLUDED.program_2, program_3 = EXCLUDED.program_3`, subject, ids[0], ids[1], ids[2])
	if err != nil {
		return fmt.Errorf("upsert battle programs: %w", err)
	}
	return nil
}
