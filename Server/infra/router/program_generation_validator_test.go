package router

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"os"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

func TestGeneratedProgramValidatorChecksSemantics(t *testing.T) {
	t.Parallel()
	valid, err := os.ReadFile("../tutorialrun/testdata/forward.json")
	if err != nil {
		t.Fatal(err)
	}
	validator := GeneratedProgramValidator{}
	if err := validator.ValidateGeneratedProgram(context.Background(), jsontext.Value(valid)); err != nil {
		t.Errorf("valid program: %v", err)
	}
	if err := validator.ValidateGeneratedProgram(context.Background(), jsontext.Value(programSource)); !errors.Is(err, usecase.ErrInvalidGeneratedProgram) {
		t.Errorf("invalid main signature: %v", err)
	}
}
