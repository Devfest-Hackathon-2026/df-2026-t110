package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

type ProgramRepository interface {
	RunProgramRepository
	Create(ctx context.Context, program domain.Program) error
	// Update changes only name and source and archives a new revision atomically.
	Update(ctx context.Context, program domain.Program) (domain.Program, error)
	GetByTutorialID(ctx context.Context, subject, tutorialID string) ([]domain.Program, error)
}

// ProgramSourceValidator checks syntax for creation and full static validity for updates.
type ProgramSourceValidator interface {
	ValidateSource(ctx context.Context, source []byte) error
	ValidateExecutableSource(ctx context.Context, source []byte) error
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

// Update uses the path ID and authenticated owner, ignoring the draft's identity.
// Updates must pass syntax, name resolution and type checking before persistence.
func (p *Programs) Update(ctx context.Context, subject, id string, draft domain.Program) (domain.Program, error) {
	if strings.TrimSpace(subject) == "" {
		return domain.Program{}, ErrUnauthenticated
	}
	program, err := p.repo.Get(ctx, id)
	if err != nil {
		return domain.Program{}, fmt.Errorf("get program to update: %w", err)
	}
	if program.GoogleSubject != subject {
		return domain.Program{}, ErrForbidden
	}
	draft.IsBattle, draft.TutorialID = program.IsBattle, program.TutorialID
	if err := draft.Validate(); err != nil {
		return domain.Program{}, fmt.Errorf("validate program: %w", err)
	}
	if err := p.validator.ValidateExecutableSource(ctx, draft.Source); err != nil {
		return domain.Program{}, fmt.Errorf("validate program source: %w", err)
	}
	draft.ID, draft.GoogleSubject = id, subject
	updated, err := p.repo.Update(ctx, draft)
	if err != nil {
		return domain.Program{}, fmt.Errorf("update program: %w", err)
	}
	return updated, nil
}

func (p *Programs) ListByTutorialID(ctx context.Context, subject, tutorialID string) ([]domain.Program, error) {
	if strings.TrimSpace(subject) == "" {
		return nil, ErrUnauthenticated
	}
	programs, err := p.repo.GetByTutorialID(ctx, subject, tutorialID)
	if err != nil {
		return nil, fmt.Errorf("list tutorial programs: %w", err)
	}
	return programs, nil
}
