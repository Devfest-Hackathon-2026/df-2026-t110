package repo

import (
	"context"
	"slices"
	"strings"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

var _ usecase.TutorialRepository = (*Tutorials)(nil)

// Tutorials keeps tutorials in memory, ordered by ID. It is read-only after
// construction and therefore safe for concurrent use without locking.
type Tutorials struct {
	tutorials map[string]domain.Tutorial
}

// NewTutorials returns a store holding the given tutorials.
func NewTutorials(tutorials ...domain.Tutorial) *Tutorials {
	r := &Tutorials{tutorials: make(map[string]domain.Tutorial, len(tutorials))}
	for _, t := range tutorials {
		r.tutorials[t.ID] = t
	}
	return r
}

func (r *Tutorials) List(_ context.Context) ([]domain.Tutorial, error) {
	tutorials := make([]domain.Tutorial, 0, len(r.tutorials))
	for _, t := range r.tutorials {
		tutorials = append(tutorials, t)
	}
	slices.SortFunc(tutorials, func(a, b domain.Tutorial) int { return strings.Compare(a.ID, b.ID) })
	return tutorials, nil
}

func (r *Tutorials) Get(_ context.Context, id string) (domain.Tutorial, error) {
	t, ok := r.tutorials[id]
	if !ok {
		return domain.Tutorial{}, usecase.ErrNotFound
	}
	return t, nil
}
