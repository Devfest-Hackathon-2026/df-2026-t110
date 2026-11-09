package router

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

func TestGeneratedProgramValidatorChecksSemantics(t *testing.T) {
	t.Parallel()
	valid, err := os.ReadFile("../tutorialrun/testdata/forward.json")
	if err != nil {
		t.Fatal(err)
	}
	const placeholder = "00000000-0000-4000-8000-000000000000"
	valid = regexp.MustCompile(`[0-9a-f-]{36}`).ReplaceAll(valid, []byte(placeholder))
	validator := GeneratedProgramValidator{}
	if got, err := validator.ValidateGeneratedProgram(context.Background(), jsontext.Value(valid)); err != nil {
		t.Errorf("valid program: %v", err)
	} else if strings.Contains(string(got), placeholder) {
		t.Errorf("validator returned provider hint IDs: %s", got)
	}
	if _, err := validator.ValidateGeneratedProgram(context.Background(), jsontext.Value(programSource)); !errors.Is(err, usecase.ErrInvalidGeneratedProgram) {
		t.Errorf("invalid main signature: %v", err)
	}
}
