package repo

import (
	"context"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo/gen/db"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ usecase.ProgramRepository = (*Programs)(nil)

type Programs struct{ queries *db.Queries }

func NewPrograms(pool *pgxpool.Pool) *Programs {
	return &Programs{queries: db.New(pool)}
}

func (r *Programs) Create(ctx context.Context, program domain.Program) error {
	if err := r.queries.CreateProgram(ctx, db.CreateProgramParams{
		ID: program.ID, GoogleSubject: program.GoogleSubject, Name: program.Name,
		Source: program.Source, IsBattle: program.IsBattle, TutorialID: textFromPtr(program.TutorialID),
	}); err != nil {
		return fmt.Errorf("insert program: %w", err)
	}
	return nil
}
