package tutorialrun_test

import (
	"context"
	"encoding/json/jsontext"
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
		name    string
		source  []byte
		cleared bool
		wantErr error
	}{
		{name: "goal and replay recording", source: source, cleared: true},
		{name: "finished without reaching goal", source: []byte(strings.Replace(string(source), `"forward"`, `"pick_up"`, 1))},
		{name: "step limit", source: loop},
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
				History []jsontext.Value `json:"history"`
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
			if strings.Contains(string(result), `"type":"picolang"`) {
				t.Error("recording contains disabled execution logs")
			}
		})
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := tutorialrun.NewExecutor().Execute(canceled, domain.Program{Source: source}, lesson, 0); !errors.Is(err, context.Canceled) {
		t.Errorf("canceled Execute: %v", err)
	}
}
