package repo

import (
	"context"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo/gen/db"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ usecase.MatchRepository = (*Matches)(nil)
var _ usecase.MatchOpponentRepository = (*Matches)(nil)

type Matches struct{ pool *pgxpool.Pool }

func NewMatches(pool *pgxpool.Pool) *Matches { return &Matches{pool: pool} }

func (r *Matches) Draw(ctx context.Context, challenger string) (domain.BattleTeam, error) {
	rows, err := db.New(r.pool).DrawBattleOpponent(ctx, db.DrawBattleOpponentParams{
		Challenger: challenger, RobotCount: domain.BattleRobotCount,
	})
	if err != nil {
		return domain.BattleTeam{}, fmt.Errorf("draw battle opponent: %w", err)
	}
	if len(rows) == 0 {
		return domain.BattleTeam{}, usecase.ErrNoOpponent
	}
	team := domain.BattleTeam{User: domain.User{GoogleSubject: rows[0].GoogleSubject, Name: &rows[0].Name.String}}
	for _, row := range rows {
		team.Programs = append(team.Programs, domain.Program{
			ID: row.ID, GoogleSubject: row.GoogleSubject, Name: row.ProgramName, Source: row.Source, IsBattle: true,
		})
	}
	return team, nil
}

func (r *Matches) Create(ctx context.Context, record domain.MatchRecord) (domain.Match, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.Match{}, fmt.Errorf("generate match ID: %w", err)
	}
	runID, err := uuid.NewV7()
	if err != nil {
		return domain.Match{}, fmt.Errorf("generate match run ID: %w", err)
	}
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		queries := db.New(tx)
		if err := queries.CreateMatchRun(ctx, db.CreateMatchRunParams{
			ID: pgtype.UUID{Bytes: runID, Valid: true}, GoogleSubject: record.GoogleSubject,
			FieldID: record.FieldID, Result: record.Result,
		}); err != nil {
			return fmt.Errorf("insert match run: %w", err)
		}
		if err := queries.CreateMatch(ctx, db.CreateMatchParams{
			ID: pgtype.UUID{Bytes: id, Valid: true}, RunID: pgtype.UUID{Bytes: runID, Valid: true},
			OpponentGoogleSubject: record.OpponentSubject, OpponentName: record.OpponentName, SelectedLayoutID: record.SelectedLayoutID,
		}); err != nil {
			return fmt.Errorf("insert match: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.Match{}, err
	}
	return domain.Match{
		ID: id.String(), RunID: runID.String(), FieldID: record.FieldID,
		SelectedLayoutID: record.SelectedLayoutID, OpponentName: record.OpponentName,
	}, nil
}
