package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
)

func TestInteractivePrimitivePrints(t *testing.T) {
	t.Parallel()
	input := "p1 print_unit ()\np2 print_rotation right\np1 print_distance_target enemy\np2 print_terrain {\"type\":\"land\",\"kind\":{\"type\":\"plain\"},\"hasLuggage\":true,\"territory\":\"own\"}\np1 done\np2 done\n"
	var output bytes.Buffer
	if err := run(io.NopCloser(strings.NewReader(input)), &output, 2, 100, board.AllStoppedRules{}); err != nil {
		t.Error(err)
	}
	for _, want := range []string{"p1 says ()", "p2 says right", "p1 says enemy", `p2 says {"type":"land","kind":{"type":"plain"},"hasLuggage":true,"territory":"own"}`, "p1 => ()", "p2 => ()", "finished at tick 2"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("missing %q in output: %s", want, output.String())
		}
	}
	if strings.Contains(output.String(), "invalid command") {
		t.Errorf("primitive print command was rejected: %s", output.String())
	}
}
