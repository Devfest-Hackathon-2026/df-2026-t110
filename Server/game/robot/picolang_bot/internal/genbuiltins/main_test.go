package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testSchema = "../../../../../../Schema/picolang/header_v0/header.schema.yaml"

func TestValidateHeader(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		input string
		valid bool
	}{
		{"primitive", `{"f":{"params":["bool"],"returnType":"unit"}}`, true},
		{"nested independent parameters", `{"f":{"typeParameters":["A","B"],"params":[{"kind":"stack","elementType":{"kind":"option","elementType":{"kind":"type_parameter","name":"A"}}},{"kind":"type_parameter","name":"B"}],"returnType":{"kind":"type_parameter","name":"A"}}}`, true},
		{"old array", `[{"name":"f","signature":{"params":[],"returnType":"bool"}}]`, false},
		{"tags", `{"f":{"params":[],"returnType":"bool","tags":["movement"]}}`, false},
		{"invalid type", `{"f":{"params":[],"returnType":"number"}}`, false},
		{"empty name", `{"":{"params":[],"returnType":"bool"}}`, false},
		{"missing params", `{"f":{"returnType":"bool"}}`, false},
		{"duplicate declaration", `{"f":{"typeParameters":["T","T"],"params":[],"returnType":"bool"}}`, false},
		{"undeclared return", `{"f":{"params":[],"returnType":{"kind":"type_parameter","name":"T"}}}`, false},
		{"undeclared nested argument", `{"f":{"typeParameters":["A"],"params":[{"kind":"stack","elementType":{"kind":"option","elementType":{"kind":"type_parameter","name":"B"}}}],"returnType":"unit"}}`, false},
		{"duplicate function", `{"f":{"params":[],"returnType":"bool"},"f":{"params":[],"returnType":"integer"}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := validate([]byte(tc.input), testSchema)
			if (err == nil) != tc.valid {
				t.Errorf("valid = %t, error = %v", tc.valid, err)
			}
		})
	}
}

func TestGenerationAndCheck(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	input, output := filepath.Join(dir, "header.json"), filepath.Join(dir, "builtins.go")
	if err := os.WriteFile(input, []byte(`{"z":{"params":[],"returnType":"unit"},"a":{"typeParameters":["X"],"params":[{"kind":"type_parameter","name":"X"}],"returnType":{"kind":"option","elementType":{"kind":"type_parameter","name":"X"}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(input, testSchema, output, false); err != nil {
		t.Fatal(err)
	}
	if err := run(input, testSchema, output, true); err != nil {
		t.Error(err)
	}
	first, err := os.ReadFile(output) //nolint:gosec // Output is a generated file inside t.TempDir.
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(string(first), `"a":`) > strings.Index(string(first), `"z":`) {
		t.Error("function names are not sorted")
	}
	if err := run(input, testSchema, output, false); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(output) //nolint:gosec // Output is a generated file inside t.TempDir.
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Error("generation is not deterministic")
	}
	if err := os.WriteFile(output, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(input, testSchema, output, true); err == nil {
		t.Error("check accepted stale output")
	}
	if err := os.WriteFile(input, []byte(`[]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(input, testSchema, output, false); err == nil {
		t.Error("generation accepted invalid input")
	}
	unchanged, err := os.ReadFile(output) //nolint:gosec // Output is a generated file inside t.TempDir.
	if err != nil {
		t.Fatal(err)
	}
	if string(unchanged) != "stale" {
		t.Error("invalid input overwrote the output")
	}
}
