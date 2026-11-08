package router

import (
	"context"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/parse"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

var _ usecase.ProgramSourceValidator = ProgramSourceValidator{}

type ProgramSourceValidator struct{}

func (ProgramSourceValidator) ValidateSource(ctx context.Context, source []byte) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("validate program source context: %w", err)
	}
	if _, err := parse.Parse(source); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrInvalidProgram, err)
	}
	return nil
}
