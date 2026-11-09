package tutorialrun_test

import (
	"context"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/tutorialrun"
)

func TestExecuteTutorial(t *testing.T) {
	t.Parallel()
	lessons, err := repo.NewTextbooks(t.Context(), os.DirFS("../repo/testdata/tutorials"))
	if err != nil {
		t.Fatal(err)
	}
	lesson, err := lessons.Get(t.Context(), "tutorial-test")
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile("testdata/forward.json")
	if err != nil {
		t.Fatal(err)
	}
	loop, err := os.ReadFile("testdata/loop.json")
	if err != nil {
		t.Fatal(err)
	}
	wait := func(turns int64, goal bool) []byte {
		text := strings.Replace(string(source), `"print_integer"`, `"wait"`, 1)
		text = strings.Replace(text, "9223372036854775807", fmt.Sprint(turns), 1)
		if !goal {
			text = strings.Replace(text, `"forward"`, `"pick_up"`, 1)
		}
		return []byte(text)
	}
	for _, tt := range []struct {
		name      string
		source    []byte
		cleared   bool
		wantErr   error
		endReason string
		reason    string
		tick      int
	}{
		{name: "goal and replay recording", source: source, cleared: true, endReason: "interrupted"},
		{name: "finished without reaching goal", source: []byte(strings.Replace(string(source), `"forward"`, `"pick_up"`, 1)), endReason: "completed"},
		{name: "step limit", source: loop, endReason: "step_limit"},
		{name: "stop before limit", source: wait(998, false), endReason: "completed", reason: "all_stopped", tick: 999},
		{name: "wait at limit", source: wait(1000, true), endReason: "interrupted", reason: "turn_limit", tick: 1000},
		{name: "wait beyond limit", source: wait(1001, true), endReason: "interrupted", reason: "turn_limit", tick: 1000},
		{name: "very long wait", source: wait(9223372036854775807, true), endReason: "interrupted", reason: "turn_limit", tick: 1000},
		{name: "goal before limit", source: wait(998, true), cleared: true, endReason: "interrupted", reason: "condition_met", tick: 999},
		{name: "goal at limit", source: wait(999, true), cleared: true, endReason: "interrupted", reason: "condition_met", tick: 1000},
		{name: "invalid program", source: []byte(`{}`), wantErr: domain.ErrInvalidProgram},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := tutorialrun.NewExecutor().Execute(t.Context(), domain.Program{Source: tt.source}, lesson, 0)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Execute: %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil || err != nil {
				return
			}
			var decoded struct {
				History []struct {
					Type  string `json:"type"`
					Tick  int    `json:"tick"`
					Event struct {
						Kind   string `json:"kind"`
						Reason string `json:"reason"`
					} `json:"event"`
				} `json:"history"`
				Summary struct {
					Tick    int    `json:"tick"`
					Cleared bool   `json:"cleared"`
					Reason  string `json:"reason"`
				} `json:"summary"`
			}
			if err := json.Unmarshal(result, &decoded); err != nil {
				t.Errorf("decode result: %v", err)
				return
			}
			if decoded.Summary.Cleared != tt.cleared || decoded.Summary.Reason == "" || len(decoded.History) == 0 {
				t.Errorf("result = %s", result)
			}
			if tt.reason != "" && (decoded.Summary.Reason != tt.reason || decoded.Summary.Tick != tt.tick) {
				t.Errorf("summary = %+v, want reason %s at tick %d", decoded.Summary, tt.reason, tt.tick)
			}
			if !strings.Contains(string(result), `"type":"field"`) {
				t.Error("recording missing field events")
			}
			for _, kind := range []string{"current_node", "memory_allocate", "variable_init", "variable_drop", "execution_finished"} {
				if !strings.Contains(string(result), `"kind":"`+kind+`"`) {
					t.Errorf("recording missing %s events", kind)
				}
			}
			endings := 0
			lastFieldTick := 0
			for _, event := range decoded.History {
				if event.Type == "field" {
					lastFieldTick = event.Tick
					if event.Tick > decoded.Summary.Tick {
						t.Errorf("history tick %d exceeds final tick %d", event.Tick, decoded.Summary.Tick)
					}
				}
				if event.Type == "picolang" && event.Event.Kind == "execution_finished" {
					endings++
					if event.Event.Reason != tt.endReason {
						t.Errorf("execution end reason = %s, want %s", event.Event.Reason, tt.endReason)
					}
				}
			}
			if endings != 1 {
				t.Errorf("execution_finished count = %d, want 1", endings)
			}
			if lastFieldTick != decoded.Summary.Tick {
				t.Errorf("last field tick = %d, want %d", lastFieldTick, decoded.Summary.Tick)
			}
		})
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := tutorialrun.NewExecutor().Execute(canceled, domain.Program{Source: source}, lesson, 0); !errors.Is(err, context.Canceled) {
		t.Errorf("canceled Execute: %v", err)
	}
}
