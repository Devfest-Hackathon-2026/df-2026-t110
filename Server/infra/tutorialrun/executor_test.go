package tutorialrun_test

import (
	"context"
	json "encoding/json/v2"
	"errors"
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
	for _, tt := range []struct {
		name      string
		source    []byte
		cleared   bool
		wantErr   error
		endReason string
	}{
		{name: "goal and replay recording", source: source, cleared: true, endReason: "interrupted"},
		{name: "finished without reaching goal", source: []byte(strings.Replace(string(source), `"forward"`, `"pick_up"`, 1)), endReason: "completed"},
		{name: "step limit", source: loop, endReason: "step_limit"},
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
					Event struct {
						Kind   string `json:"kind"`
						Reason string `json:"reason"`
					} `json:"event"`
				} `json:"history"`
				Summary struct {
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
			if !strings.Contains(string(result), `"type":"field"`) {
				t.Error("recording missing field events")
			}
			for _, kind := range []string{"current_node", "memory_allocate", "variable_init", "variable_drop", "execution_finished"} {
				if !strings.Contains(string(result), `"kind":"`+kind+`"`) {
					t.Errorf("recording missing %s events", kind)
				}
			}
			endings := 0
			for _, event := range decoded.History {
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
		})
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := tutorialrun.NewExecutor().Execute(canceled, domain.Program{Source: source}, lesson, 0); !errors.Is(err, context.Canceled) {
		t.Errorf("canceled Execute: %v", err)
	}
}
