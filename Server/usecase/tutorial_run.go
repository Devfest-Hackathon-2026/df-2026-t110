package usecase

import (
	"context"
	"encoding/json/jsontext"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

// RunProgramRepository retrieves saved programs, including their owner.
type RunProgramRepository interface {
	Get(ctx context.Context, id string) (domain.Program, error)
}

// TutorialExecutor executes one program synchronously, including replay recording.
type TutorialExecutor interface {
	Execute(ctx context.Context, program domain.Program, tutorial domain.Tutorial, variant int) (jsontext.Value, error)
}

type RunRepository interface {
	Create(ctx context.Context, run domain.Run) (string, error)
	SaveResult(ctx context.Context, id string, result jsontext.Value) error
}

type TutorialRuns struct {
	programs  RunProgramRepository
	tutorials TutorialRepository
	executor  TutorialExecutor
	runs      RunRepository
	pick      func(int) int
}

func NewTutorialRuns(programs RunProgramRepository, tutorials TutorialRepository, executor TutorialExecutor, runs RunRepository) *TutorialRuns {
	return &TutorialRuns{programs: programs, tutorials: tutorials, executor: executor, runs: runs, pick: rand.IntN}
}

func (r *TutorialRuns) Run(ctx context.Context, subject, tutorialID, programID string) (domain.TutorialRun, error) {
	if strings.TrimSpace(subject) == "" {
		return domain.TutorialRun{}, ErrUnauthenticated
	}
	if strings.TrimSpace(tutorialID) == "" || strings.TrimSpace(programID) == "" {
		return domain.TutorialRun{}, domain.ErrInvalidProgram
	}
	tutorial, err := r.tutorials.Get(ctx, tutorialID)
	if err != nil {
		return domain.TutorialRun{}, fmt.Errorf("get tutorial: %w", err)
	}
	program, err := r.programs.Get(ctx, programID)
	if err != nil {
		return domain.TutorialRun{}, fmt.Errorf("get program: %w", err)
	}
	if program.GoogleSubject != subject {
		return domain.TutorialRun{}, ErrForbidden
	}
	if program.IsBattle || program.TutorialID == nil || *program.TutorialID != tutorialID {
		return domain.TutorialRun{}, domain.ErrInvalidProgram
	}
	if len(tutorial.Stage.Variants) == 0 {
		return domain.TutorialRun{}, fmt.Errorf("tutorial %q has no field variants", tutorialID)
	}
	variant := r.pick(len(tutorial.Stage.Variants))
	fieldID := tutorial.Stage.Variants[variant].Id
	id, err := r.runs.Create(ctx, domain.Run{GoogleSubject: subject, TutorialID: tutorialID, FieldID: fieldID, ProgramID: programID, ProgramRevision: program.Revision})
	if err != nil {
		return domain.TutorialRun{}, fmt.Errorf("create tutorial run: %w", err)
	}
	result, err := r.executor.Execute(ctx, program, tutorial, variant)
	if err != nil {
		return domain.TutorialRun{}, fmt.Errorf("execute tutorial: %w", err)
	}
	if err := r.runs.SaveResult(ctx, id, result); err != nil {
		return domain.TutorialRun{}, fmt.Errorf("save tutorial result: %w", err)
	}
	return domain.TutorialRun{ID: id, FieldID: fieldID}, nil
}
