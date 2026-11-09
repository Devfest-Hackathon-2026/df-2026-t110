package usecase

import (
	"context"
	"encoding/json/jsontext"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

type MatchUserRepository interface {
	Get(ctx context.Context, subject string) (domain.User, error)
}

type MatchOpponentRepository interface {
	// SelectOpponent chooses a user uniformly, excluding the challenger. The user's name
	// and ordered battle programs must come from the same snapshot.
	SelectOpponent(ctx context.Context, challenger string) (domain.BattleTeam, error)
}

type MatchLayoutRepository interface {
	List(ctx context.Context) ([]domain.BattleLayout, error)
}

type MatchExecutor interface {
	Execute(ctx context.Context, player, opponent []domain.Program, layout domain.BattleLayout) (jsontext.Value, error)
}

type MatchRepository interface {
	Create(ctx context.Context, match domain.MatchRecord) (domain.Match, error)
}

type MatchDeps struct {
	Users     MatchUserRepository
	Programs  RunProgramRepository
	Opponents MatchOpponentRepository
	Layouts   MatchLayoutRepository
	Executor  MatchExecutor
	Matches   MatchRepository
}

type Matches struct {
	deps MatchDeps
	pick func(int) int
}

func NewMatches(deps MatchDeps) *Matches { return &Matches{deps: deps, pick: rand.IntN} }

// Create runs the engine in the server process and saves a completed match.
// Rejected requests and failed executions do not leave pending runs behind.
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
	// Do not update the challenger's saved defense programs as a side effect.
	opponent, err := m.deps.Opponents.SelectOpponent(ctx, subject)
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
	result, err := m.deps.Executor.Execute(ctx, programs, opponent.Programs, layout)
	if err != nil {
		return domain.Match{}, fmt.Errorf("execute match: %w", err)
	}
	// TODO: Update the challenger's rating with the result: https://github.com/dvpico/Programming-Learning-AR/issues/155
	match, err := m.deps.Matches.Create(ctx, domain.MatchRecord{
		GoogleSubject: subject, OpponentSubject: opponent.User.GoogleSubject,
		OpponentName: *opponent.User.Name, FieldID: layout.FieldID, SelectedLayoutID: layout.ID, Result: result,
	})
	if err != nil {
		return domain.Match{}, fmt.Errorf("save match: %w", err)
	}
	return match, nil
}
