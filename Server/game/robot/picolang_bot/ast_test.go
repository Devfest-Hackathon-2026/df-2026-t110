package picolangbot_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Compile fixtures exercise the public API from another package. Rejected
// operations must fail in the compiler, before any interpreter code can run.
func TestASTStageBoundaries(t *testing.T) {
	t.Parallel()

	t.Run("valid stages", func(t *testing.T) {
		t.Parallel()
		cmd := exec.CommandContext(t.Context(), "go", "test", "./testdata/valid")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("valid AST stages failed to compile: %v\n%s", err, output)
		}
	})

	t.Run("mixed stages", func(t *testing.T) {
		t.Parallel()
		data, err := os.ReadFile("testdata/invalid/invalid.go")
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.CommandContext(t.Context(), "go", "test", "-gcflags=-e", "./testdata/invalid")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("mixed AST stages unexpectedly compiled")
			return
		}
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("could not invoke the compiler: %v", err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			if !strings.Contains(line, "// want-error") {
				continue
			}
			location := fmt.Sprintf("invalid.go:%d:", i+1)
			if !strings.Contains(string(output), location) {
				t.Errorf("expected a compile error at %s (%s)\n%s", location, strings.TrimSpace(line), output)
			}
		}
	})
}
