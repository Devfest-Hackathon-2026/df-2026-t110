package repo

import (
	"context"
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

var _ usecase.MatchRepository = (*Matches)(nil)
var _ usecase.MatchOpponentRepository = (*Matches)(nil)
var _ usecase.RankingRepository = (*Matches)(nil)

type Matches struct{ pool *pgxpool.Pool }

func NewMatches(pool *pgxpool.Pool) *Matches { return &Matches{pool: pool} }

func (r *Matches) SelectOpponent(ctx context.Context, challenger string, choose func([]domain.OpponentCandidate) (domain.OpponentCandidate, error)) (domain.BattleTeam, error) {
	var team domain.BattleTeam
	// Repeatable read keeps the chosen user's team as eligible as it was when listed.
	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
		queries := db.New(tx)
		rows, err := queries.ListBattleOpponentCandidates(ctx, db.ListBattleOpponentCandidatesParams{
			Challenger: challenger, RobotCount: domain.BattleRobotCount,
		})
		if err != nil {
			return fmt.Errorf("list battle opponent candidates: %w", err)
		}
		if len(rows) == 0 {
			return usecase.ErrNoOpponent
		}
		candidates := make([]domain.OpponentCandidate, len(rows))
		for i, row := range rows {
			candidates[i] = domain.OpponentCandidate{GoogleSubject: row.GoogleSubject, Rating: int(row.Rating)}
		}
		chosen, err := choose(candidates)
		if err != nil {
			return err
		}
		programs, err := queries.GetBattleTeam(ctx, chosen.GoogleSubject)
		if err != nil {
			return fmt.Errorf("get battle team: %w", err)
		}
		if len(programs) == 0 {
			return fmt.Errorf("chosen opponent %q has no battle team", chosen.GoogleSubject)
		}
		team = domain.BattleTeam{
			User:   domain.User{GoogleSubject: programs[0].GoogleSubject, Name: &programs[0].Name.String},
			Rating: int(programs[0].Rating),
		}
		for _, row := range programs {
			team.Programs = append(team.Programs, domain.Program{
				ID: row.ID, Revision: int(row.Revision), GoogleSubject: row.GoogleSubject, Name: row.ProgramName, Source: row.Source, IsBattle: true,
			})
		}
		return nil
	})
	if err != nil {
		return domain.BattleTeam{}, err
	}
	return team, nil
}

func (r *Matches) Rating(ctx context.Context, subject string) (int, error) {
	rating, err := db.New(r.pool).GetUserRating(ctx, subject)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, usecase.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("get user rating: %w", err)
	}
	return int(rating), nil
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
		references := db.CreateRunProgramsParams{RunID: pgtype.UUID{Bytes: runID, Valid: true}}
		// Side values are part of the persisted format: challenger=0, opponent=1.
		const challengerSide, opponentSide int16 = 0, 1
		teams := []struct {
			side     int16
			programs []domain.ProgramRef
		}{
			{challengerSide, record.ChallengerPrograms}, {opponentSide, record.OpponentPrograms},
		}
		for _, team := range teams {
			if len(team.programs) > domain.BattleRobotCount {
				return domain.ErrInvalidProgram
			}
			for index, program := range team.programs {
				revision, err := programRevisionToDB(program)
				if err != nil {
					return err
				}
				references.Sides = append(references.Sides, team.side)
				references.RobotIndexes = append(references.RobotIndexes, int16(index))
				references.ProgramIds = append(references.ProgramIds, program.ID)
				references.Revisions = append(references.Revisions, revision)
			}
		}
		if err := queries.CreateRunPrograms(ctx, references); err != nil {
			return fmt.Errorf("insert match program revisions: %w", err)
		}
		after, err := queries.AddUserRating(ctx, db.AddUserRatingParams{
			GoogleSubject: record.GoogleSubject, Delta: int64(record.RatingDelta),
		})
		if err != nil {
			return fmt.Errorf("update challenger rating: %w", err)
		}
		// Store the ratings actually replaced, which differ from record.ChallengerRating
		// when another result of the challenger was applied during this match.
		if err := queries.CreateMatch(ctx, db.CreateMatchParams{
			ID: pgtype.UUID{Bytes: id, Valid: true}, RunID: pgtype.UUID{Bytes: runID, Valid: true},
			OpponentGoogleSubject: record.OpponentSubject, OpponentName: record.OpponentName,
			Outcome:        string(record.Outcome),
			OpponentRating: int64(record.OpponentRating),
			RatingBefore:   int64(after) - int64(record.RatingDelta),
			RatingAfter:    int64(after),
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
		OpponentName: record.OpponentName,
	}, nil
}

func (r *Matches) Ranking(ctx context.Context) ([]domain.RankingEntry, error) {
	rows, err := db.New(r.pool).ListRanking(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ranking: %w", err)
	}
	entries := make([]domain.RankingEntry, len(rows))
	for i, row := range rows {
		entries[i] = domain.RankingEntry{
			Rank: int(row.Rank), Name: row.Name, Rating: int(row.Rating),
			Wins: int(row.Wins), Losses: int(row.Losses), Draws: int(row.Draws),
		}
	}
	return entries, nil
}
