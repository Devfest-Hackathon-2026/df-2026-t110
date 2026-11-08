package robot_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	"github.com/dvpico/Programming-Learning-AR/Server/game/recorder"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter"
)

const picolangSource = `{
  "picolangVersion": 0, "picolang": {"v0": {"program": {"main": {
    "kind": "function", "hintId": "00000000-0000-4000-8000-000000000001",
    "params": [], "returnType": "unit", "body": {
      "kind": "block", "hintId": "00000000-0000-4000-8000-000000000002", "stmts": [
        {"kind": "declare", "hintId": "00000000-0000-4000-8000-000000000003", "type": "bool",
         "ident": {"kind": "ident", "hintId": "00000000-0000-4000-8000-000000000004", "name": "moved"},
         "value": {"kind": "call", "hintId": "00000000-0000-4000-8000-000000000005", "func": "forward", "args": []}},
        {"kind": "call", "hintId": "00000000-0000-4000-8000-000000000006", "func": "print_bool", "args": [
          {"kind": "ident", "hintId": "00000000-0000-4000-8000-000000000007", "name": "moved"}]},
        {"kind": "return", "hintId": "00000000-0000-4000-8000-000000000008", "value": {
          "kind": "literal", "hintId": "00000000-0000-4000-8000-000000000009", "type": "unit", "value": null}}
      ]
    }
  }}}}
}`

func picolangLimits() interpreter.Limits {
	return interpreter.Limits{MaxSteps: 100, MaxMemoryBytes: 64 << 10, MaxCallDepth: 10}
}

func TestPicolangBotConstructorErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		source string
		limits interpreter.Limits
		stage  bot.AnalysisStage
		code   bot.DiagnosticCode
	}{
		{"parse", `{`, picolangLimits(), "", ""},
		{"variable binding", strings.Replace(picolangSource, `"name": "moved"`, `"name": "other"`, 1), picolangLimits(), bot.IndexStage, bot.CodeUndefinedVariable},
		{"undefined function", strings.Replace(picolangSource, `"func": "forward"`, `"func": "unknown"`, 1), picolangLimits(), bot.IndexStage, bot.CodeUndefinedFunction},
		{"type check", strings.Replace(picolangSource, `"type": "bool"`, `"type": "integer"`, 1), picolangLimits(), bot.TypeCheckStage, bot.CodeTypeMismatch},
		{"main signature", strings.Replace(picolangSource, `"returnType": "unit"`, `"returnType": "integer"`, 1), picolangLimits(), bot.TypeCheckStage, bot.CodeInvalidMain},
		{"VM construction", picolangSource, interpreter.Limits{}, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r, err := robot.NewPicolangBot([]byte(tc.source), 42, tc.limits, true)
			if err == nil || r != nil {
				t.Errorf("constructor returned %v, %v for invalid input", r, err)
			}
			if tc.stage == "" {
				return
			}
			var diagnostics bot.Diagnostics
			if !errors.As(err, &diagnostics) {
				t.Errorf("constructor lost semantic diagnostics: %v", err)
				return
			}
			found := false
			for _, diagnostic := range diagnostics {
				if diagnostic.Stage == tc.stage && diagnostic.Code == tc.code && diagnostic.Function == "main" && diagnostic.HintID != "" {
					found = true
				}
			}
			if !found {
				t.Errorf("missing %s/%s diagnostic with source: %v", tc.stage, tc.code, err)
			}
		})
	}
}

func TestPicolangBotRunsThroughHub(t *testing.T) {
	t.Parallel()
	source := []byte(picolangSource)
	r, err := robot.NewPicolangBot(source, 42, picolangLimits(), true)
	if err != nil {
		t.Errorf("NewPicolangBot: %v", err)
		return
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	if events := r.Collect(); len(events) != 0 {
		t.Errorf("constructor executed the program: %#v", events)
	}
	// The bot owns its AST and remains usable after the caller reuses the buffer.
	for i := range source {
		source[i] = '?'
	}
	plain := board.NewLandTerrain(board.PlainLand{}, false, board.NoTeam)
	config, err := board.NewConfig([][]board.Terrain{{plain, plain}}, map[protocol.RobotID]board.RobotState{
		"robot": board.NewRobotState(board.NewPosition(0, 0), board.East, 3),
	}, 10, board.AllStoppedRules{})
	if err != nil {
		t.Fatalf("prepare board config: %v", err)
	}
	field, err := board.New(config)
	if err != nil {
		t.Fatalf("prepare board: %v", err)
	}
	h, err := hub.New(field, []hub.Participant{{ID: "robot", Robot: r}})
	if err != nil {
		t.Fatalf("prepare hub: %v", err)
	}
	storage := &recorder.Recorder{}
	report, err := h.Run(hub.Options{MaxResumes: 10, Recorder: storage})
	if err != nil || report.Outcome("robot") != (hub.CompletedOutcome{}) || report.Resumes() != 3 {
		t.Errorf("Run = %+v, %v", report, err)
	}
	boundResult, completed := false, 0
	for _, event := range storage.Events {
		if event, ok := event.(gameevent.RobotEvent); ok && event.InterpreterID == "robot" {
			switch event := event.Event.(type) {
			case robotevent.VariableInitEvent:
				if event.HintID == "00000000-0000-4000-8000-000000000004" && event.Value == (values.BoolValue{Bool: true}) {
					boundResult = true
				}
			case robotevent.ExecutionFinishedEvent:
				if event.Reason == robotevent.EndCompleted {
					completed++
				}
			}
		}
	}
	if !boundResult || completed != 1 {
		t.Errorf("binding result or completion event missing: binding=%t, completions=%d", boundResult, completed)
	}
}

func TestPicolangBotDefersExecutionUntilStart(t *testing.T) {
	t.Parallel()
	limits := picolangLimits()
	limits.MaxSteps = 1
	r, err := robot.NewPicolangBot([]byte(picolangSource), 42, limits, true)
	if err != nil {
		t.Errorf("constructor spent the execution budget: %v", err)
		return
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	output, err := r.Start()
	if !errors.Is(err, interpreter.ErrStepLimit) || output.Step != nil {
		t.Errorf("Start did not propagate the VM error: %+v, %v", output, err)
	}
	finished := 0
	for _, event := range output.Events {
		if end, ok := event.(robotevent.ExecutionFinishedEvent); ok && end.Reason == robotevent.EndStepLimit {
			finished++
		}
	}
	if finished != 1 {
		t.Errorf("got %d resource-limit events, want one", finished)
	}
}
