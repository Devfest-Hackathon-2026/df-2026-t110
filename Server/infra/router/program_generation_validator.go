package router

import (
	"context"
	"encoding/json/jsontext"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/index"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/parse"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/typecheck"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

// GeneratedProgramValidator checks the same Picolang stages used by the robot.
type GeneratedProgramValidator struct{}

var _ usecase.GeneratedProgramValidator = GeneratedProgramValidator{}

func (GeneratedProgramValidator) ValidateGeneratedProgram(ctx context.Context, source jsontext.Value) (jsontext.Value, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	source, err := assignGeneratedHintIDs(source)
	if err != nil {
		return nil, err
	}
	parsed, err := parse.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", usecase.ErrInvalidGeneratedProgram, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	indexed, err := index.Index(parsed)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", usecase.ErrInvalidGeneratedProgram, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := typecheck.TypeCheck(indexed); err != nil {
		return nil, fmt.Errorf("%w: %w", usecase.ErrInvalidGeneratedProgram, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return source, nil
}
