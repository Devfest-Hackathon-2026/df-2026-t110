package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

type BattleProgramRepository interface {
	Get(ctx context.Context, subject string) ([]string, error)
	Set(ctx context.Context, subject string, ids []string) error
}

type BattlePrograms struct {
	repo     BattleProgramRepository
	programs RunProgramRepository
}

func NewBattlePrograms(repo BattleProgramRepository, programs RunProgramRepository) *BattlePrograms {
	return &BattlePrograms{repo: repo, programs: programs}
}

// Set preserves the robot order and permits repeated program IDs.
func (b *BattlePrograms) Set(ctx context.Context, subject string, ids []string) error {
	if strings.TrimSpace(subject) == "" {
		return ErrUnauthenticated
	}
	if len(ids) != domain.BattleRobotCount {
		return domain.ErrInvalidProgram
	}
	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			return domain.ErrInvalidProgram
		}
		program, err := b.programs.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("get battle program: %w", err)
		}
		if program.GoogleSubject != subject {
			return ErrForbidden
		}
		if !program.IsBattle {
			return domain.ErrInvalidProgram
		}
	}
	if err := b.repo.Set(ctx, subject, ids); err != nil {
		return fmt.Errorf("save battle programs: %w", err)
	}
	return nil
}

// Get returns the three saved slots, or ErrNotFound when no settings exist.
func (b *BattlePrograms) Get(ctx context.Context, subject string) ([]string, error) {
	if strings.TrimSpace(subject) == "" {
		return nil, ErrUnauthenticated
	}
	ids, err := b.repo.Get(ctx, subject)
	if err != nil {
		return nil, fmt.Errorf("get battle programs: %w", err)
	}
	return ids, nil
}
