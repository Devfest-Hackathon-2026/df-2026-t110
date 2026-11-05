package usecase

import (
	"context"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

// TutorialRepository stores tutorials.
type TutorialRepository interface {
	List(ctx context.Context) ([]domain.Tutorial, error)
	// Get returns the tutorial or ErrNotFound.
	Get(ctx context.Context, id string) (domain.Tutorial, error)
}

// Tutorials serves tutorials to learners.
type Tutorials struct {
	repo TutorialRepository
}

func NewTutorials(repo TutorialRepository) *Tutorials {
	return &Tutorials{repo: repo}
}

func (t *Tutorials) List(ctx context.Context) ([]domain.Tutorial, error) {
	return t.repo.List(ctx)
}

func (t *Tutorials) Get(ctx context.Context, id string) (domain.Tutorial, error) {
	return t.repo.Get(ctx, id)
}
