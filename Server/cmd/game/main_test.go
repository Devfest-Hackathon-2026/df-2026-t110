package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func TestRenderTerrainKinds(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	err := render(&output, board.Snapshot{
		Tick: 7,
		Terrain: [][]values.TerrainValue{{
			values.LandTerrain{Kind: values.PlainLand{}}, values.LandTerrain{Kind: values.GoalLand{}}, values.LandTerrain{Kind: values.CheckpointLand{Order: 2}},
			values.LandTerrain{Kind: values.PlainLand{}, HasLuggage: true}, values.RockTerrain{}, values.SeaTerrain{},
		}},
	})
	if err != nil {
		t.Error(err)
	}
	if want := "\ntick 7\n  .  G  C  L  #  ~\n"; output.String() != want {
		t.Errorf("got terrain display %q, want %q", output.String(), want)
	}
}

func TestInteractiveDemo(t *testing.T) {
	t.Parallel()
	input := "p1 drill_durability\np1 forward\np2 forward\np1 forward\np2 forward\np1 done\np2 done\n"
	var output bytes.Buffer
	if err := run(io.NopCloser(strings.NewReader(input)), &output, 1000, 10000); err != nil {
		t.Error(err)
	}
	for _, want := range []string{"tick 0", "tick 1", "tick 2", "p1 => 3", "p1 => true", "p2 => true", "p1 => false", "p2 => false", "p1: (1,0) east", "p2: (3,0) west", "p1: completed", "p2: completed"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("missing %q in output:\n%s", want, output.String())
		}
	}
}

func TestEOFFinishesPendingTick(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"", "p1 forward\n"} {
		var output bytes.Buffer
		if err := run(io.NopCloser(strings.NewReader(input)), &output, 1000, 10000); err != nil {
			t.Error(err)
		}
		want := "finished at tick 0"
		if input != "" {
			want = "finished at tick 1"
			if !strings.Contains(output.String(), "p1 => true") {
				t.Error("pending action did not finish after EOF")
			}
		}
		if !strings.Contains(output.String(), want) || !strings.Contains(output.String(), "p2: completed") {
			t.Errorf("unexpected EOF output:\n%s", output.String())
		}
	}
}

func TestCLITickLimit(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	err := run(io.NopCloser(strings.NewReader("p1 forward\np2 forward\np1 forward\np2 forward\n")), &output, 1, 100)
	if !errors.Is(err, board.ErrTickLimit) || strings.Contains(output.String(), "tick 2") {
		t.Errorf("tick limit not enforced: %v\n%s", err, output.String())
	}
}
