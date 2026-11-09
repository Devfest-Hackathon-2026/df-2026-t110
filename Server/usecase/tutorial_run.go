package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

// ProgramGetter retrieves saved programs, including their owner.
type ProgramGetter interface {
	Get(ctx context.Context, id string) (domain.Program, error)
}

// Run executes the subject's program on a randomly chosen field of the tutorial
// and returns the run whose result is fetched by its ID.
func (t *Tutorials) Run(ctx context.Context, subject, tutorialID, programID string) (domain.TutorialRun, error) {
	if t.programs == nil || t.executor == nil {
		return domain.TutorialRun{}, ErrRunUnavailable
	}
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
	ref := program.Ref()
	run := domain.Run{GoogleSubject: subject, TutorialID: tutorialID, FieldID: fieldID, Program: &ref}
	id, err := t.executor.Execute(ctx, run, domain.TutorialJob{Program: program, Tutorial: tutorial, Variant: variant})
	if err != nil {
		return domain.TutorialRun{}, fmt.Errorf("execute tutorial: %w", err)
	}
	return domain.TutorialRun{ID: id, FieldID: fieldID}, nil
}
