package usecase

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidGenerationRequest = errors.New("invalid program generation request")
	ErrInvalidGeneratedProgram  = errors.New("invalid generated program")
)

const (
	DefaultGenerationAttempts = 2
	DefaultGenerationTimeout  = 30 * time.Second
)

// GenerationFields retrieves the field definition visible to the authenticated user.
type GenerationFields interface {
	GetGenerationField(ctx context.Context, subject, fieldID string) (jsontext.Value, error)
}

// ProgramGenerator must request structured JSON output from its provider and enforce
// MaxTokens as an upper bound on the tokens charged for the call.
type ProgramGenerator interface {
	GenerateProgram(ctx context.Context, request ProgramGenerationInput) (jsontext.Value, error)
}

type ProgramGenerationInput struct {
	Prompt          string
	Field           jsontext.Value
	IsBattle        bool
	PreviousOutput  jsontext.Value
	ValidationError string
	MaxTokens       int
}

// GeneratedProgramValidator validates the complete program, including its syntax
// and semantics. Invalid output must wrap ErrInvalidGeneratedProgram; operational
// failures must be returned separately and are not sent to the generator.
//
// TODO: Back this with the language engine so semantics are checked as well.
// Until it is available, an implementation that only checks that the program is
// grammatically valid is sufficient.
type GeneratedProgramValidator interface {
	ValidateGeneratedProgram(ctx context.Context, program jsontext.Value) error
}

type ProgramGenerationDeps struct {
	Fields      GenerationFields
	Quota       GenerationQuota
	Generator   ProgramGenerator
	Validator   GeneratedProgramValidator
	MaxTokens   int
	MaxAttempts int
	Timeout     time.Duration
}

type ProgramGeneration struct{ deps ProgramGenerationDeps }

type ProgramGenerationRequest struct {
	FieldID  string
	Prompt   string
	IsBattle bool
}

func NewProgramGeneration(deps ProgramGenerationDeps) *ProgramGeneration {
	if deps.MaxAttempts <= 0 {
		deps.MaxAttempts = DefaultGenerationAttempts
	}
	if deps.Timeout <= 0 {
		deps.Timeout = DefaultGenerationTimeout
	}
	return &ProgramGeneration{deps: deps}
}

// Generate returns a validated draft without saving it. subject must come from a
// verified Google identity, never directly from an untrusted request.
func (g *ProgramGeneration) Generate(ctx context.Context, subject, fieldID, prompt string) (jsontext.Value, error) {
	if strings.TrimSpace(fieldID) == "" {
		return nil, ErrInvalidGenerationRequest
	}
	return g.GenerateWithOptions(ctx, subject, ProgramGenerationRequest{FieldID: fieldID, Prompt: prompt})
}

// GenerateWithOptions also supports requests without a field. Such requests
// receive no stage context, but still receive the selected tutorial/battle rules.
func (g *ProgramGeneration) GenerateWithOptions(ctx context.Context, subject string, request ProgramGenerationRequest) (jsontext.Value, error) {
	if strings.TrimSpace(subject) == "" {
		return nil, ErrUnauthenticated
	}
	if strings.TrimSpace(request.Prompt) == "" || len(request.Prompt) > 4096 {
		return nil, ErrInvalidGenerationRequest
	}
	if g.deps.Fields == nil || g.deps.Quota == nil || g.deps.Generator == nil || g.deps.Validator == nil || g.deps.MaxTokens <= 0 {
		return nil, errors.New("program generation is not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, g.deps.Timeout)
	defer cancel()
	field := jsontext.Value(`{}`)
	if strings.TrimSpace(request.FieldID) != "" {
		var err error
		field, err = g.deps.Fields.GetGenerationField(ctx, subject, request.FieldID)
		if err != nil {
			return nil, fmt.Errorf("get generation field: %w", err)
		}
	}
	if !field.IsValid() {
		return nil, errors.New("invalid field data")
	}
	input := ProgramGenerationInput{Prompt: request.Prompt, Field: field, IsBattle: request.IsBattle, MaxTokens: g.deps.MaxTokens}
	for attempt := 0; attempt < g.deps.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := g.deps.Quota.Reserve(ctx, subject, input.MaxTokens); err != nil {
			return nil, fmt.Errorf("reserve generation quota: %w", err)
		}
		output, err := g.deps.Generator.GenerateProgram(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("generate program: %w", err)
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// Structured output constrains the provider's response, but this output
		// is still untrusted input; it is checked before it reaches the validator.
		if !output.IsValid() {
			err = fmt.Errorf("%w: output is not valid JSON", ErrInvalidGeneratedProgram)
		} else {
			err = g.deps.Validator.ValidateGeneratedProgram(ctx, output)
		}
		if err == nil {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return output, nil
		}
		if !errors.Is(err, ErrInvalidGeneratedProgram) {
			return nil, fmt.Errorf("validate generated program: %w", err)
		}
		input.PreviousOutput, input.ValidationError = output, err.Error()
	}
	return nil, fmt.Errorf("%w: attempts exhausted", ErrInvalidGeneratedProgram)
}
