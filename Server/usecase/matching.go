package usecase

import (
	"context"
	"math/rand/v2"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

// SubmissionRepository lists the programs submitted to a field.
type SubmissionRepository interface {
	// LatestByField returns, for each user who submitted to the field, the program
	// that user updated last. It returns ErrNotFound if the field does not exist.
	LatestByField(ctx context.Context, fieldID string) ([]domain.Submission, error)
}

// LayoutRepository lists the candidate layouts of a field.
type LayoutRepository interface {
	// LayoutIDs returns ErrNotFound if the field does not exist.
	LayoutIDs(ctx context.Context, fieldID string) ([]string, error)
}

// Matching draws the opponent and layout of ranked matches.
type Matching struct {
	submissions SubmissionRepository
	layouts     LayoutRepository
	pick        func(n int) int
}

func NewMatching(submissions SubmissionRepository, layouts LayoutRepository) *Matching {
	return &Matching{submissions: submissions, layouts: layouts, pick: rand.IntN}
}

// Draw chooses a layout and a random opponent other than the challenger.
// It returns ErrNoOpponent when nobody else has submitted to the field.
func (m *Matching) Draw(ctx context.Context, fieldID, challenger string) (domain.Pairing, error) {
	layoutIDs, err := m.layouts.LayoutIDs(ctx, fieldID)
	if err != nil {
		return domain.Pairing{}, err
	}
	if len(layoutIDs) == 0 {
		return domain.Pairing{}, ErrNotFound
	}
	submissions, err := m.submissions.LatestByField(ctx, fieldID)
	if err != nil {
		return domain.Pairing{}, err
	}
	opponent, err := domain.SelectOpponent(challenger, submissions, m.pick)
	if err != nil {
		return domain.Pairing{}, err
	}
	return domain.Pairing{LayoutID: layoutIDs[m.pick(len(layoutIDs))], Opponent: opponent}, nil
}
