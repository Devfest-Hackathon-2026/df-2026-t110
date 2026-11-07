package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
)

func TestRenderTerrainKinds(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	err := render(&output, board.Snapshot{
		Tick: 7,
		Terrain: [][]board.Terrain{{
			board.NewLandTerrain(board.PlainLand{}, false, 0),
			board.NewLandTerrain(board.GoalLand{}, false, 0),
			board.NewLandTerrain(board.NewCheckpointLand(2), false, 0),
			board.NewLandTerrain(board.PlainLand{}, true, 0),
			board.RockTerrain{},
			board.SeaTerrain{},
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
	input := "p1 drill_durability\np1 search_now\np1 forward\np2 search_now\np2 search_front\np2 forward\np1 search_now\np1 forward\np2 forward\np1 done\np2 done\n"
	var output bytes.Buffer
	if err := run(io.NopCloser(strings.NewReader(input)), &output, 1000, 10000, board.AllStoppedRules{}); err != nil {
		t.Error(err)
	}
	for _, want := range []string{"tick 0", "tick 1", "tick 2", "p1 => 3", "p1 => terrain(land, territory=own)", "p2 => terrain(land, territory=own)", "p2 => terrain(land)", "p1 => terrain(land)", "finished at tick 2", "p1 => true", "p2 => true", "p1 => false", "p2 => false", "p1: (1,0) east", "p2: (3,0) west", "p1: completed", "p2: completed"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("missing %q in output:\n%s", want, output.String())
		}
	}
}

func TestInteractivePrintInteger(t *testing.T) {
	t.Parallel()

	input := "p1 print_integer -42\np2 print_integer 9223372036854775807\np1 print_bool false\np2 print_bool true\np1 done\np2 done\n"
	want := []string{"p1 says -42", "p2 says 9223372036854775807", "p1 says false", "p2 says true", "p1 => ()", "p2 => ()", "finished at tick 2", "p1: completed", "p2: completed"}
	var output bytes.Buffer

	err := run(io.NopCloser(strings.NewReader(input)), &output, 2, 100, board.AllStoppedRules{})

	if err != nil {
		t.Error(err)
	}
	for _, text := range want {
		if !strings.Contains(output.String(), text) {
			t.Errorf("missing %q in output: %s", text, output.String())
		}
	}
	if strings.Contains(output.String(), "invalid command") {
		t.Errorf("print command was rejected: %s", output.String())
	}
}

func TestInteractiveDrill(t *testing.T) {
	t.Parallel()

	input := "p1 forward\np2 done\np1 forward\np1 rotate right\np1 drill\np1 drill\np1 drill_durability\np1 search_front\np1 forward\np1 done\n"
	want := []string{"p1 => false", "p1 => 2", "p1 => terrain(land)", "p1: (2,1) south", "finished at tick 6"}
	var output bytes.Buffer

	err := run(io.NopCloser(strings.NewReader(input)), &output, 6, 100, board.AllStoppedRules{})

	if err != nil {
		t.Error(err)
	}
	for _, line := range want {
		if !strings.Contains(output.String(), line) {
			t.Errorf("missing %q in output: %s", line, output.String())
		}
	}
	if strings.Contains(output.String(), "invalid command") {
		t.Errorf("drill was rejected: %s", output.String())
	}
}

func TestInteractiveLuggage(t *testing.T) {
	t.Parallel()

	input := "p1 forward\np2 done\np1 rotate right\np1 pick_up\np1 search_front\np1 release\np1 search_front\np1 done\n"
	want := []string{"p1: (1,0) south carrying luggage", "p1 => terrain(land)", "p1 => terrain(land, luggage)", "finished at tick 4"}
	var output bytes.Buffer

	err := run(io.NopCloser(strings.NewReader(input)), &output, 4, 100, board.AllStoppedRules{})

	if err != nil {
		t.Error(err)
	}
	for _, line := range want {
		if !strings.Contains(output.String(), line) {
			t.Errorf("missing %q in output: %s", line, output.String())
		}
	}
}

func TestInteractiveWait(t *testing.T) {
	t.Parallel()

	input := "p1 wait abc\np1 wait 0\np1 wait 3\np2 wait 1\np2 done\np1 done\n"
	want := []string{"invalid command for p1", "p1 => ()", "p2 => ()", "\ntick 1\n", "\ntick 2\n", "\ntick 3\n", "finished at tick 3"}
	var output bytes.Buffer

	err := run(io.NopCloser(strings.NewReader(input)), &output, 3, 100, board.AllStoppedRules{})

	if err != nil {
		t.Error(err)
	}
	for _, line := range want {
		if !strings.Contains(output.String(), line) {
			t.Errorf("missing %q in output: %s", line, output.String())
		}
	}
	if strings.Count(output.String(), "invalid command") != 1 {
		t.Errorf("unexpected command errors: %s", output.String())
	}
}

func TestInteractiveDistance(t *testing.T) {
	t.Parallel()

	input := "p1 distance invalid\np1 distance sea\np1 distance robot\np1 done\np2 distance enemy\np2 done\n"
	want := []string{"p1 => 5", "p1 => 4", "p2 => 4", "finished at tick 0"}
	var output bytes.Buffer

	err := run(io.NopCloser(strings.NewReader(input)), &output, 1, 100, board.AllStoppedRules{})

	if err != nil || strings.Count(output.String(), "invalid command") != 1 {
		t.Errorf("distance commands = %v, output: %s", err, output.String())
	}
	for _, line := range want {
		if !strings.Contains(output.String(), line) {
			t.Errorf("missing %q in output: %s", line, output.String())
		}
	}
}

func TestEOFFinishesPendingTick(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"", "p1 forward\n"} {
		var output bytes.Buffer
		if err := run(io.NopCloser(strings.NewReader(input)), &output, 1000, 10000, board.AllStoppedRules{}); err != nil {
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

func TestCLIBattleEndsDuringWait(t *testing.T) {
	t.Parallel()

	input := "p1 wait 100\np2 wait 100\n"
	newRules := func() board.CompletionRules {
		t.Helper()
		rules, err := completionRules(3, 1, 1)
		if err != nil {
			t.Fatal(err)
		}
		return rules
	}
	battle, safety := newRules(), newRules()
	var output bytes.Buffer

	err := run(io.NopCloser(strings.NewReader(input)), &output, 5, 100, battle)

	if err != nil || !strings.Contains(output.String(), "finished at tick 3") || !strings.Contains(output.String(), "battle ended (turn_limit): team 1 = 0, team 2 = 0, draw") {
		t.Errorf("battle failed to end at the scheduled turn: %v\n%s", err, output.String())
	}
	output.Reset()
	err = run(io.NopCloser(strings.NewReader(input)), &output, 2, 100, safety)
	if !errors.Is(err, board.ErrTickLimit) || !strings.Contains(output.String(), "finished at tick 2") || strings.Contains(output.String(), "battle ended") {
		t.Errorf("safety limit did not abort the battle: %v\n%s", err, output.String())
	}
}

func TestCLITickLimit(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	err := run(io.NopCloser(strings.NewReader("p1 forward\np2 forward\np1 forward\np2 forward\n")), &output, 1, 100, board.AllStoppedRules{})
	if !errors.Is(err, board.ErrTickLimit) || strings.Contains(output.String(), "tick 2") {
		t.Errorf("tick limit not enforced: %v\n%s", err, output.String())
	}
}
