package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
)

func TestInteractiveDemo(t *testing.T) {
	input := "p1 drill_durability\np1 forward\np2 forward\np1 forward\np2 forward\np1 done\np2 done\n"
	var output bytes.Buffer
	if err := run(t.Context(), io.NopCloser(strings.NewReader(input)), &output, 1000, 10000); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"tick 0", "tick 1", "tick 2", "p1 => 3", "p1 => true", "p2 => true", "p1 => false", "p2 => false", "p1: (1,0) right", "p2: (3,0) left", "p1: completed", "p2: completed"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("missing %q in output:\n%s", want, output.String())
		}
	}
}

func TestEOFFinishesPendingTick(t *testing.T) {
	for _, input := range []string{"", "p1 forward\n"} {
		var output bytes.Buffer
		if err := run(t.Context(), io.NopCloser(strings.NewReader(input)), &output, 1000, 10000); err != nil {
			t.Fatal(err)
		}
		want := "finished at tick 0"
		if input != "" {
			want = "finished at tick 1"
			if !strings.Contains(output.String(), "p1 => true") {
				t.Fatal("pending action did not finish after EOF")
			}
		}
		if !strings.Contains(output.String(), want) || !strings.Contains(output.String(), "p2: completed") {
			t.Fatalf("unexpected EOF output:\n%s", output.String())
		}
	}
}

func TestCLITickLimit(t *testing.T) {
	var output bytes.Buffer
	err := run(t.Context(), io.NopCloser(strings.NewReader("p1 forward\np2 forward\np1 forward\np2 forward\n")), &output, 1, 100)
	if !errors.Is(err, hub.ErrTickLimit) || strings.Contains(output.String(), "tick 2") {
		t.Fatalf("tick limit not enforced: %v\n%s", err, output.String())
	}
}
