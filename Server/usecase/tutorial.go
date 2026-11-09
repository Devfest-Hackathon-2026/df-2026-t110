package usecase

import (
	"context"
	"math/rand/v2"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

// TutorialRepository stores tutorials.
type TutorialRepository interface {
	List(ctx context.Context) ([]domain.Tutorial, error)
	// Get returns the tutorial or ErrNotFound.
	Get(ctx context.Context, id string) (domain.Tutorial, error)
}

// Tutorials serves tutorials to learners and runs their programs.
type Tutorials struct {
	tutorials TutorialRepository
	programs  ProgramGetter
	executor  *Executor
	pick      func(int) int
}

// NewTutorials takes nil programs and executor when no database is configured;
// tutorials can then be read but not run.
func NewTutorials(tutorials TutorialRepository, programs ProgramGetter, executor *Executor) *Tutorials {
	return &Tutorials{tutorials: tutorials, programs: programs, executor: executor, pick: rand.IntN}
}

func (t *Tutorials) List(ctx context.Context) ([]domain.Tutorial, error) {
	return t.tutorials.List(ctx)
}

func (t *Tutorials) Get(ctx context.Context, id string) (domain.Tutorial, error) {
	return t.tutorials.Get(ctx, id)
}
