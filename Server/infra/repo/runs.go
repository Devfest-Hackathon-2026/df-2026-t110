package repo

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo/gen/db"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ usecase.RunRepository = (*Runs)(nil)
var _ usecase.RunResultRepository = (*Runs)(nil)

type Runs struct{ queries *db.Queries }

func NewRuns(pool *pgxpool.Pool) *Runs { return &Runs{queries: db.New(pool)} }

func (r *Runs) Create(ctx context.Context, run domain.Run) (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate run ID: %w", err)
	}
	if err := r.queries.CreateRun(ctx, db.CreateRunParams{
		ID: pgtype.UUID{Bytes: id, Valid: true}, GoogleSubject: run.GoogleSubject,
		TutorialID: run.TutorialID, FieldID: run.FieldID, ProgramID: run.ProgramID,
	}); err != nil {
		return "", fmt.Errorf("insert run: %w", err)
	}
	return id.String(), nil
}

func (r *Runs) SaveResult(ctx context.Context, id string, result jsontext.Value) error {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return usecase.ErrNotFound
	}
	rows, err := r.queries.SaveRunResult(ctx, db.SaveRunResultParams{ID: pgtype.UUID{Bytes: parsed, Valid: true}, Result: result})
	if err != nil {
		return fmt.Errorf("update run result: %w", err)
	}
	if rows == 0 {
		return usecase.ErrNotFound
	}
	return nil
}

func (r *Runs) Get(ctx context.Context, id string) (domain.Run, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return domain.Run{}, usecase.ErrNotFound
	}
	row, err := r.queries.GetRun(ctx, pgtype.UUID{Bytes: parsed, Valid: true})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Run{}, usecase.ErrNotFound
	}
	if err != nil {
		return domain.Run{}, fmt.Errorf("get run: %w", err)
	}
	return domain.Run{ID: id, GoogleSubject: row.GoogleSubject, TutorialID: row.TutorialID, FieldID: row.FieldID, ProgramID: row.ProgramID, Result: row.Result}, nil
}
