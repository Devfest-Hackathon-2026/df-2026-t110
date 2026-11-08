package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

type ProgramRepository interface {
	Create(ctx context.Context, program domain.Program) error
}

// ProgramSourceValidator checks the versioned source syntax. Drafts need not
// pass name resolution or type checking until execution.
type ProgramSourceValidator interface {
	ValidateSource(ctx context.Context, source []byte) error
}

type Programs struct {
	repo      ProgramRepository
	validator ProgramSourceValidator
}

func NewPrograms(repo ProgramRepository, validator ProgramSourceValidator) *Programs {
	return &Programs{repo: repo, validator: validator}
}

// Create always assigns a fresh ID and takes ownership from the authenticated
// subject, ignoring any client-supplied identity on the draft.
func (p *Programs) Create(ctx context.Context, subject string, draft domain.Program) (domain.Program, error) {
	if strings.TrimSpace(subject) == "" {
		return domain.Program{}, ErrUnauthenticated
	}
	if err := draft.Validate(); err != nil {
		return domain.Program{}, fmt.Errorf("validate program: %w", err)
	}
	if err := p.validator.ValidateSource(ctx, draft.Source); err != nil {
		return domain.Program{}, fmt.Errorf("validate program source: %w", err)
	}
	id, err := newRandomID()
	if err != nil {
		return domain.Program{}, fmt.Errorf("generate program ID: %w", err)
	}
	draft.ID, draft.GoogleSubject = id, subject
	if err := p.repo.Create(ctx, draft); err != nil {
		return domain.Program{}, fmt.Errorf("create program: %w", err)
	}
	return draft, nil
}
