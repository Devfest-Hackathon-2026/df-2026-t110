package usecase

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

// ProgramGetter retrieves saved programs, including their owner.
type ProgramGetter interface {
	Get(ctx context.Context, id string) (domain.Program, error)
}

type TutorialRuns struct {
	programs  ProgramGetter
	tutorials TutorialRepository
	executor  *Executor
	pick      func(int) int
}

func NewTutorialRuns(programs ProgramGetter, tutorials TutorialRepository, executor *Executor) *TutorialRuns {
	return &TutorialRuns{programs: programs, tutorials: tutorials, executor: executor, pick: rand.IntN}
}

func (t *TutorialRuns) Run(ctx context.Context, subject, tutorialID, programID string) (domain.TutorialRun, error) {
	if strings.TrimSpace(subject) == "" {
		return domain.TutorialRun{}, ErrUnauthenticated
	}
	if strings.TrimSpace(tutorialID) == "" || strings.TrimSpace(programID) == "" {
		return domain.TutorialRun{}, domain.ErrInvalidProgram
	}
	tutorial, err := t.tutorials.Get(ctx, tutorialID)
	if err != nil {
		return domain.TutorialRun{}, fmt.Errorf("get tutorial: %w", err)
	}
	program, err := t.programs.Get(ctx, programID)
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
	variant := t.pick(len(tutorial.Stage.Variants))
	fieldID := tutorial.Stage.Variants[variant].Id
	run := domain.Run{GoogleSubject: subject, TutorialID: tutorialID, FieldID: fieldID, ProgramID: programID}
	id, err := t.executor.Execute(ctx, run, domain.TutorialJob{Program: program, Tutorial: tutorial, Variant: variant})
	if err != nil {
		return domain.TutorialRun{}, fmt.Errorf("execute tutorial: %w", err)
	}
	return domain.TutorialRun{ID: id, FieldID: fieldID}, nil
}
