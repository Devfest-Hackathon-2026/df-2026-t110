package usecase

import (
	"context"
	"encoding/json/jsontext"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

type RunRepository interface {
	Create(ctx context.Context, run domain.Run) (string, error)
	SaveResult(ctx context.Context, id string, result jsontext.Value) error
}

type MatchResultRepository interface {
	// Complete saves the result of the run together with the rated match.
	Complete(ctx context.Context, runID string, record domain.MatchRecord) error
}

// TutorialSimulator runs a program on a tutorial variant without touching storage.
type TutorialSimulator interface {
	Simulate(ctx context.Context, program domain.Program, tutorial domain.Tutorial, variant int) (jsontext.Value, error)
}

// MatchSimulator runs two teams on a layout without touching storage.
type MatchSimulator interface {
	Simulate(ctx context.Context, player, opponent []domain.Program, layout domain.BattleLayout) (domain.MatchResult, error)
}

// Executor creates runs, simulates their jobs and saves the results. It keeps
// no state of its own; runs and their results live in the repositories.
type Executor struct {
	runs              RunRepository
	matchResults      MatchResultRepository
	tutorialSimulator TutorialSimulator
	matchSimulator    MatchSimulator
}

func NewExecutor(runs RunRepository, matchResults MatchResultRepository, tutorialSimulator TutorialSimulator, matchSimulator MatchSimulator) *Executor {
	return &Executor{runs: runs, matchResults: matchResults, tutorialSimulator: tutorialSimulator, matchSimulator: matchSimulator}
}

// Execute returns the run ID, which callers use to fetch the result. It creates
// the run before simulating, so a failed simulation leaves a pending run behind.
func (e *Executor) Execute(ctx context.Context, run domain.Run, job domain.Job) (string, error) {
	id, err := e.runs.Create(ctx, run)
	if err != nil {
		return "", fmt.Errorf("create run: %w", err)
	}
	switch job := job.(type) {
	case domain.TutorialJob:
		result, err := e.tutorialSimulator.Simulate(ctx, job.Program, job.Tutorial, job.Variant)
		if err != nil {
			return "", fmt.Errorf("simulate tutorial: %w", err)
		}
		if err := e.runs.SaveResult(ctx, id, result); err != nil {
			return "", fmt.Errorf("save tutorial result: %w", err)
		}
	case domain.MatchJob:
		result, err := e.matchSimulator.Simulate(ctx, job.Player, job.Opponent, job.Layout)
		if err != nil {
			return "", fmt.Errorf("simulate match: %w", err)
		}
		if err := e.matchResults.Complete(ctx, id, job.Entry.Record(result)); err != nil {
			return "", fmt.Errorf("save match: %w", err)
		}
	default:
		return "", fmt.Errorf("unsupported job %T", job)
	}
	return id, nil
}
