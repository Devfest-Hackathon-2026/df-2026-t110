package usecase

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/google/uuid"
)

type MatchUserRepository interface {
	Get(ctx context.Context, subject string) (domain.User, error)
}

type MatchOpponentRepository interface {
	// SelectOpponent passes choose the users other than the challenger who have a display
	// name and a complete battle team, and returns the team of the chosen user. The
	// candidates and the chosen user's name, rating and ordered battle programs must come
	// from the same snapshot. It returns ErrNoOpponent when there is no candidate.
	SelectOpponent(ctx context.Context, challenger string, choose func([]domain.OpponentCandidate) (domain.OpponentCandidate, error)) (domain.BattleTeam, error)
}

type MatchLayoutRepository interface {
	List(ctx context.Context) ([]domain.BattleLayout, error)
}

type MatchRepository interface {
	// Rating returns the challenger's latest rating. It is read without a lock: a match
	// is rated from the ratings at matching time.
	Rating(ctx context.Context, subject string) (int, error)
}

type MatchDeps struct {
	Users     MatchUserRepository
	Programs  ProgramGetter
	Opponents MatchOpponentRepository
	Layouts   MatchLayoutRepository
	Executor  *Executor
	Matches   MatchRepository
}

type Matches struct {
	deps  MatchDeps
	pick  func(int) int
	draw  func() float64
	newID func() (uuid.UUID, error)
}

func NewMatches(deps MatchDeps) *Matches {
	return &Matches{deps: deps, pick: rand.IntN, draw: rand.Float64, newID: uuid.NewV7}
}

// Create matches the challenger with an opponent and hands the match to the
// executor. A failed simulation leaves a pending run without a match.
func (m *Matches) Create(ctx context.Context, subject string, ids []string) (domain.Match, error) {
	if strings.TrimSpace(subject) == "" {
		return domain.Match{}, ErrUnauthenticated
	}
	if len(ids) != domain.BattleRobotCount {
		return domain.Match{}, domain.ErrInvalidProgram
	}
	user, err := m.deps.Users.Get(ctx, subject)
	if err != nil {
		return domain.Match{}, fmt.Errorf("get challenger: %w", err)
	}
	if user.Name == nil || strings.TrimSpace(*user.Name) == "" {
		return domain.Match{}, ErrForbidden
	}
	programs := make([]domain.Program, len(ids))
	for i, id := range ids {
		if strings.TrimSpace(id) == "" {
			return domain.Match{}, domain.ErrInvalidProgram
		}
		program, err := m.deps.Programs.Get(ctx, id)
		if err != nil {
			return domain.Match{}, fmt.Errorf("get challenger program: %w", err)
		}
		if program.GoogleSubject != subject {
			return domain.Match{}, ErrForbidden
		}
		if !program.IsBattle {
			return domain.Match{}, domain.ErrInvalidProgram
		}
		programs[i] = program
	}
	rating, err := m.deps.Matches.Rating(ctx, subject)
	if err != nil {
		return domain.Match{}, fmt.Errorf("get challenger rating: %w", err)
	}
	// Do not update the challenger's saved defense programs as a side effect.
	opponent, err := m.deps.Opponents.SelectOpponent(ctx, subject, func(candidates []domain.OpponentCandidate) (domain.OpponentCandidate, error) {
		return domain.SelectRatedOpponent(subject, rating, candidates, m.draw)
	})
	if err != nil {
		return domain.Match{}, fmt.Errorf("select opponent: %w", err)
	}
	if opponent.User.GoogleSubject == subject || opponent.User.Name == nil || len(opponent.Programs) != domain.BattleRobotCount {
		return domain.Match{}, fmt.Errorf("invalid opponent snapshot")
	}
	layouts, err := m.deps.Layouts.List(ctx)
	if err != nil {
		return domain.Match{}, fmt.Errorf("list battle layouts: %w", err)
	}
	if len(layouts) == 0 {
		return domain.Match{}, ErrNotFound
	}
	layout := layouts[m.pick(len(layouts))]
	id, err := m.newID()
	if err != nil {
		return domain.Match{}, fmt.Errorf("generate match ID: %w", err)
	}
	entry := domain.MatchEntry{
		ID: id.String(), GoogleSubject: subject, OpponentSubject: opponent.User.GoogleSubject, OpponentName: *opponent.User.Name,
		ChallengerRating: rating, OpponentRating: opponent.Rating,
	}
	run := domain.Run{GoogleSubject: subject, FieldID: layout.FieldID}
	runID, err := m.deps.Executor.Execute(ctx, run, domain.MatchJob{Entry: entry, Player: programs, Opponent: opponent.Programs, Layout: layout})
	if err != nil {
		return domain.Match{}, fmt.Errorf("execute match: %w", err)
	}
	return domain.Match{ID: entry.ID, RunID: runID, FieldID: layout.FieldID, OpponentName: entry.OpponentName}, nil
}
