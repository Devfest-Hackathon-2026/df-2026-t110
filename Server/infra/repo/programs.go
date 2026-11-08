package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo/gen/db"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ usecase.ProgramRepository = (*Programs)(nil)
var _ usecase.RunProgramRepository = (*Programs)(nil)

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

func (r *Programs) Get(ctx context.Context, id string) (domain.Program, error) {
	row, err := r.queries.GetProgram(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Program{}, usecase.ErrNotFound
	}
	if err != nil {
		return domain.Program{}, fmt.Errorf("get program: %w", err)
	}
	program := domain.Program{ID: row.ID, GoogleSubject: row.GoogleSubject, Name: row.Name, Source: row.Source, IsBattle: row.IsBattle}
	if row.TutorialID.Valid {
		program.TutorialID = &row.TutorialID.String
	}
	return program, nil
}
