package router

import (
	"context"
	"encoding/json/jsontext"
	"errors"
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

// ValidateExecutableSource uses the same static analysis pipeline as execution.
func (ProgramSourceValidator) ValidateExecutableSource(ctx context.Context, source []byte) error {
	_, err := (GeneratedProgramValidator{}).ValidateGeneratedProgram(ctx, jsontext.Value(source))
	if errors.Is(err, usecase.ErrInvalidGeneratedProgram) {
		return fmt.Errorf("%w: %w", domain.ErrInvalidProgram, err)
	}
	return err
}
