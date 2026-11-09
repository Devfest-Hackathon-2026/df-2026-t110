package main

import (
	"bufio"
	"bytes"
	json "encoding/json/v2"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	wire "github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/history"
)

func call(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out, stderr bytes.Buffer
	err := execute(t.Context(), args, &out, &stderr)
	return out.String(), err
}
func generateBots(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "bots")
	if _, err := call(t, "generate", "--out", dir, "--count", "8", "--seed", "42"); err != nil {
		t.Fatal(err)
	}
	return dir
}
func TestGenerateAndReproducibleBattle(t *testing.T) {
	t.Parallel()
	bots := generateBots(t)
	data := readTestFile(t, filepath.Join(bots, "manifest.json"))
	var manifest struct {
		Seed     uint64          `json:"seed"`
		Programs []manifestEntry `json:"programs"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Seed != 42 || len(manifest.Programs) != 8 {
		t.Errorf("manifest=%+v", manifest)
	}
	root := t.TempDir()
	var histories [][]byte
	for _, name := range []string{"first", "second"} {
		dir := filepath.Join(root, name)
		out, err := call(t, "run", "--player", filepath.Join(bots, "bot-000003-courier.json"), "--out", dir, "--seed", "42")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "winner=") {
			t.Errorf("output=%s", out)
		}
		data := readTestFile(t, filepath.Join(dir, "result.json"))
		var r result
		if err := json.Unmarshal(data, &r); err != nil {
			t.Fatal(err)
		}
		if r.Ticks != r.Layout.Turns || len(r.Robots) != 6 || r.Error != "" {
			t.Errorf("result=%+v", r)
		}
		for _, robot := range r.Robots {
			if robot.Status == "failed" {
				t.Errorf("robot failed: %+v", robot)
			}
		}
		history := readTestFile(t, filepath.Join(dir, "history.jsonl"))
		histories = append(histories, history)
		scanner := bufio.NewScanner(bytes.NewReader(history))
		count, lastTick := 0, 0
		for scanner.Scan() {
			var event wire.FieldEvent
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				t.Fatal(err)
			}
			if count == 0 && (event.State.Initial == nil || len(event.State.Initial.Robots) != 6) {
				t.Error("missing initial state with six robots")
			}
			if event.Tick < lastTick {
				t.Error("ticks are out of order")
			}
			count++
			lastTick = event.Tick
		}
		if err := scanner.Err(); err != nil {
			t.Fatal(err)
		}
		if count < 201 || lastTick != 200 {
			t.Errorf("history count=%d lastTick=%d", count, lastTick)
		}
		if _, err := os.Stat(filepath.Join(dir, "team-2-robot-1.json")); err != nil {
			t.Error("generated opponent source not saved:", err)
		}
	}
	if !bytes.Equal(histories[0], histories[1]) {
		t.Error("same seed produced different history")
	}
}
func TestCLIValidationAndRuntimeErrors(t *testing.T) {
	t.Parallel()
	bots := generateBots(t)
	courier := filepath.Join(bots, "bot-000003-courier.json")
	for _, args := range [][]string{
		{"generate", "--count", "0"}, {"generate", "--profile", "unknown"}, {"run"},
		{"run", "--player", courier, "--player", courier},
		{"run", "--player", courier, "--layout", "missing"},
		{"run", "--player", courier, "--max-steps", "0"},
		{"generate", "--out", bots}, {"generate", "unexpected"}, {"unknown"},
	} {
		if _, err := call(t, args...); err == nil {
			t.Errorf("accepted invalid args %v", args)
		}
	}
	out := filepath.Join(t.TempDir(), "failed")
	_, err := call(t, "run", "--player", courier, "--max-steps", "5", "--out", out)
	if !errors.Is(err, errRobotFailed) {
		t.Errorf("runtime failure=%v", err)
	}
	data := readTestFile(t, filepath.Join(out, "result.json"))
	var r result
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	if r.Robots[0].Status != "failed" || r.Robots[0].Error == "" {
		t.Errorf("runtime failure omitted: %+v", r.Robots[0])
	}
	invalid := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(invalid, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = call(t, "run", "--player", courier, "--opponent", invalid, "--out", filepath.Join(t.TempDir(), "bad"))
	if err == nil {
		t.Error("invalid opponent accepted")
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func TestRecorderFailure(t *testing.T) {
	t.Parallel()
	err := (lineRecorder{ctx: t.Context(), writer: brokenWriter{}}).Record(gameevent.FieldEvent{Event: fieldevent.FieldEvent{State: fieldevent.UpdateState{RemovedRobotIDs: []protocol.RobotID{"robot"}}}})
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Errorf("record error=%v", err)
	}
}

func TestRunExplicitTeamsAndCustomLayout(t *testing.T) {
	t.Parallel()
	bots := generateBots(t)
	layouts, err := loadLayouts(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	layoutPath := filepath.Join(root, "layout.json")
	if err := writeJSON(layoutPath, layouts[1]); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "battle")
	files := []string{"bot-000001-sentry.json", "bot-000002-wanderer.json", "bot-000003-courier.json", "bot-000004-miner.json", "bot-000005-sentry.json", "bot-000006-wanderer.json"}
	args := []string{"run", "--layout-file", layoutPath, "--turns", "12", "--out", out}
	for i, file := range files {
		flag := "--player"
		if i >= 3 {
			flag = "--opponent"
		}
		args = append(args, flag, filepath.Join(bots, file))
	}
	if _, err := call(t, args...); err != nil {
		t.Fatal(err)
	}
	var r result
	if err := json.Unmarshal(readTestFile(t, filepath.Join(out, "result.json")), &r); err != nil {
		t.Fatal(err)
	}
	if r.Ticks != 12 || r.Layout.FieldID != layouts[1].FieldID {
		t.Errorf("ticks=%d layout=%s", r.Ticks, r.Layout.FieldID)
	}
	for i, robot := range r.Robots {
		if filepath.Base(robot.Source) != files[i] {
			t.Errorf("slot %d source=%s", i, robot.Source)
		}
		snapshot := readTestFile(t, filepath.Join(out, robot.ID+".json"))
		original := readTestFile(t, robot.Source)
		if !bytes.Equal(snapshot, original) {
			t.Errorf("slot %d saved source differs", i)
		}
	}
}

func readTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // Test callers only read fixtures inside temporary directories.
	if err != nil {
		t.Fatal(err)
	}
	return data
}
