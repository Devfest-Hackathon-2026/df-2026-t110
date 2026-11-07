package interpreter

import (
	json "encoding/json/v2"
	"errors"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	"github.com/dvpico/Programming-Learning-AR/Server/game/recorder"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/function"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
)

func expectLimitFailure(t *testing.T, v *VM, output robotmsg.RobotOutput, err, want error, hint bot.HintID, base uint64) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Errorf("failure = %v, want %v", err, want)
	}
	var runtimeErr *function.RuntimeError
	if !errors.As(err, &runtimeErr) || runtimeErr.HintID != hint {
		t.Errorf("failure lost its source %s: %v", hint, err)
	}
	reasons := map[error]robotevent.ExecutionEndReason{
		ErrStepLimit: robotevent.EndStepLimit, ErrMemoryLimit: robotevent.EndMemoryLimit, ErrCallDepthLimit: robotevent.EndCallDepthLimit,
	}
	expectExecutionEnd(t, output.Events, reasons[want], hint)
	if output.Step != nil {
		t.Errorf("failed VM returned a successful step: %#v", output.Step)
	}
	assertClean(t, v, base)
	if pending := v.Collect(); len(pending) != 0 {
		t.Errorf("failure events were not delivered: %#v", pending)
	}
	if _, err := v.Next(values.UnitValue{}); !errors.Is(err, ErrState) {
		t.Errorf("failed VM accepted Next: %v", err)
	}
	if _, err := json.Marshal(output.Events); err != nil {
		t.Errorf("failure events cannot be encoded: %v", err)
	}
}

func TestResourceLimitsMustBePositive(t *testing.T) {
	t.Parallel()
	b := &builder{}
	program := checked(t, map[bot.Name]fn{"main": b.function(nil, bot.UnitType{}, b.ret(b.unit()))})
	for _, limits := range []Limits{
		{},
		{MaxSteps: 1, MaxMemoryBytes: 1},
		{MaxSteps: 1, MaxCallDepth: 1},
		{MaxMemoryBytes: 1, MaxCallDepth: 1},
	} {
		if vm, err := New(program, 0, limits); err == nil || vm != nil {
			t.Errorf("accepted invalid limits %+v: %v, %v", limits, vm, err)
		}
	}
}

func TestStepLimitStopsLoopWithoutActions(t *testing.T) {
	t.Parallel()
	b := &builder{}
	body := b.block()
	loop := bot.While[bot.Name, bot.Untyped]{Source: b.source(), Condition: bot.LiteralExpr[bot.Name, bot.Untyped]{Source: b.source(), Value: bot.BoolLiteral{Value: true}}, Body: body}
	program := checked(t, map[bot.Name]fn{"main": b.function(nil, bot.UnitType{}, loop, b.ret(b.unit()))})
	limits := testLimits()
	limits.MaxSteps = 10
	v := newLimitedVM(t, program, 0, limits)
	base := v.memory.MemoryUsage()
	output, err := v.Start()
	expectLimitFailure(t, v, output, err, ErrStepLimit, body.HintID, base)
}

func TestStepLimitPersistsAcrossActions(t *testing.T) {
	t.Parallel()
	b := &builder{}
	local := b.declare("kept", bot.IntegerType{}, b.integer(1))
	second := b.call("print_integer", b.integer(2))
	program := checked(t, map[bot.Name]fn{"main": b.function(nil, bot.UnitType{}, local, b.call("print_integer", b.integer(1)), second, b.ret(b.unit()))})
	limits := testLimits()
	limits.MaxSteps = 4
	v := newLimitedVM(t, program, 0, limits)
	base := v.memory.MemoryUsage()
	output, err := v.Start()
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.PrintIntegerAction{Integer: 1}})
	output, err = v.Next(values.UnitValue{})
	expectLimitFailure(t, v, output, err, ErrStepLimit, second.HintID, base)
	foundDrop := false
	for _, e := range output.Events {
		if drop, ok := e.(robotevent.VariableDropEvent); ok && drop.HintID == local.Ident.HintID {
			foundDrop = true
		}
	}
	if !foundDrop {
		t.Error("failure did not deliver the local variable's cleanup event")
	}
}

func TestMemoryLimitIncludesTemporaryBuiltinResults(t *testing.T) {
	t.Parallel()
	b := &builder{}
	add := b.call("add", b.integer(1), b.integer(2))
	program := checked(t, map[bot.Name]fn{"main": b.function(nil, bot.UnitType{}, add, b.ret(b.unit()))})
	var store memory.Store
	for _, value := range []int64{1, 2} {
		if _, err := store.Add(memory.Integer{Value: value}); err != nil {
			t.Fatalf("prepare memory budget: %v", err)
		}
	}
	limits := testLimits()
	limits.MaxMemoryBytes = store.MemoryUsage()
	v := newLimitedVM(t, program, 0, limits)
	base := v.memory.MemoryUsage()
	output, err := v.Start()
	expectLimitFailure(t, v, output, err, ErrMemoryLimit, add.HintID, base)
}

func TestMemoryLimitOnActionResult(t *testing.T) {
	t.Parallel()
	b := &builder{}
	search := b.call("search_front")
	program := checked(t, map[bot.Name]fn{"main": b.function(nil, bot.UnitType{}, search, b.ret(b.unit()))})
	limits := testLimits()
	var store memory.Store
	limits.MaxMemoryBytes = store.MemoryUsage()
	v := newLimitedVM(t, program, 0, limits)
	base := v.memory.MemoryUsage()
	output, err := v.Start()
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.SearchFrontAction{}})
	// Invalid board results remain retryable and do not spend the memory budget.
	if _, err := v.Next(values.BoolValue{Bool: true}); err == nil {
		t.Error("accepted an invalid action result")
	}
	output, err = v.Next(values.LandTerrain{Kind: values.PlainLand{}})
	expectLimitFailure(t, v, output, err, ErrMemoryLimit, search.HintID, base)
}

func TestResourceLimitsAllowExactBudgets(t *testing.T) {
	t.Parallel()
	b := &builder{}
	program := checked(t, map[bot.Name]fn{"main": b.function(nil, bot.UnitType{}, b.ret(b.unit()))})
	var store memory.Store
	if _, err := store.Add(memory.Unit{}); err != nil {
		t.Fatalf("prepare memory budget: %v", err)
	}
	limits := Limits{MaxSteps: 2, MaxMemoryBytes: store.MemoryUsage(), MaxCallDepth: 1}
	v := newLimitedVM(t, program, 0, limits)
	base := v.memory.MemoryUsage()
	output, err := v.Start()
	expectOutput(t, output, err, robotmsg.DoneStep{})
	assertClean(t, v, base)
}

func TestMemoryLimitBelowSharedRoot(t *testing.T) {
	t.Parallel()
	b := &builder{}
	main := b.function(nil, bot.UnitType{}, b.ret(b.unit()))
	program := checked(t, map[bot.Name]fn{"main": main})
	limits := testLimits()
	limits.MaxMemoryBytes = 1
	v := newLimitedVM(t, program, 0, limits)
	base := v.memory.MemoryUsage()
	output, err := v.Start()
	expectLimitFailure(t, v, output, err, ErrMemoryLimit, main.HintID, base)
}

func TestResourceFailuresReachHubLikeDivisionByZero(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"steps", "memory", "call depth", "division by zero"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			b := &builder{}
			local := b.declare("kept", bot.IntegerType{}, b.integer(1))
			recurse := b.call("recurse")
			calleeValue := b.integer(2)
			callee := b.function(nil, bot.UnitType{}, b.declare("inner", bot.IntegerType{}, calleeValue), recurse, b.ret(b.unit()))
			failure := b.call("recurse")
			limits := testLimits()
			var want error
			var hint bot.HintID
			switch kind {
			case "steps":
				limits.MaxSteps, want, hint = 6, ErrStepLimit, recurse.HintID
			case "memory":
				var store memory.Store
				if _, err := store.Add(memory.Integer{Value: 1}); err != nil {
					t.Fatalf("prepare memory budget: %v", err)
				}
				limits.MaxMemoryBytes, want, hint = store.MemoryUsage(), ErrMemoryLimit, bot.SourceOf(calleeValue, bot.Source{}).HintID
			case "call depth":
				limits.MaxCallDepth, want, hint = 2, ErrCallDepthLimit, recurse.HintID
			case "division by zero":
				failure = b.call("div", b.integer(1), b.integer(0))
				hint = failure.HintID
			}
			program := checked(t, map[bot.Name]fn{"main": b.function(nil, bot.UnitType{}, local, failure, b.ret(b.unit())), "recurse": callee})
			v := newLimitedVM(t, program, 0, limits)
			base := v.memory.MemoryUsage()
			healthy := newVM(t, checked(t, map[bot.Name]fn{"main": b.function(nil, bot.UnitType{}, b.call("forward"), b.ret(b.unit()))}), 0)
			plain := board.NewLandTerrain(board.PlainLand{}, false, 0)
			config, err := board.NewConfig([][]board.Terrain{{plain, plain}, {plain, plain}}, map[protocol.RobotID]board.RobotState{
				"failed":  board.NewRobotState(board.NewPosition(0, 0), board.East, 3),
				"healthy": board.NewRobotState(board.NewPosition(0, 1), board.East, 3),
			}, 10, board.AllStoppedRules{})
			if err != nil {
				t.Fatalf("prepare config: %v", err)
			}
			field, err := board.New(config)
			if err != nil {
				t.Fatalf("prepare board: %v", err)
			}
			h, err := hub.New(field, []hub.Participant{{ID: "failed", Robot: v}, {ID: "healthy", Robot: healthy}})
			if err != nil {
				t.Fatalf("prepare hub: %v", err)
			}
			storage := &recorder.Recorder{}
			report, err := h.Run(hub.Options{MaxResumes: 10, Recorder: storage})
			if err != nil {
				t.Errorf("robot failure stopped the hub: %v", err)
			}
			outcome, ok := report.Outcome("failed").(hub.FailedOutcome)
			if !ok {
				t.Errorf("failed outcome = %#v", report.Outcome("failed"))
			}
			if want != nil && !errors.Is(outcome.Err, want) {
				t.Errorf("hub lost the resource failure: %v, want %v", outcome.Err, want)
			}
			var runtimeErr *function.RuntimeError
			if !errors.As(outcome.Err, &runtimeErr) || runtimeErr.HintID != hint {
				t.Errorf("hub lost the failure source %s: %v", hint, outcome.Err)
			}
			if report.Outcome("healthy") != (hub.CompletedOutcome{}) {
				t.Errorf("other robot did not complete: %#v", report.Outcome("healthy"))
			}
			foundDrop := false
			var endings []robotevent.ExecutionFinishedEvent
			for _, e := range storage.Events {
				if e, ok := e.(gameevent.RobotEvent); ok && e.InterpreterID == "failed" {
					if end, ok := e.Event.(robotevent.ExecutionFinishedEvent); ok {
						endings = append(endings, end)
					}
					if drop, ok := e.Event.(robotevent.VariableDropEvent); ok && drop.HintID == local.Ident.HintID {
						foundDrop = true
					}
				}
			}
			if !foundDrop {
				t.Error("hub did not record failure cleanup events")
			}
			wantReason := map[string]robotevent.ExecutionEndReason{
				"steps": robotevent.EndStepLimit, "memory": robotevent.EndMemoryLimit,
				"call depth": robotevent.EndCallDepthLimit, "division by zero": robotevent.EndDivisionByZero,
			}[kind]
			if len(endings) != 1 || endings[0].Reason != wantReason || endings[0].HintID != hint {
				t.Errorf("recorder lost the ending: %+v, want %s at %s", endings, wantReason, hint)
			}
			if _, err := json.Marshal(storage.Events); err != nil {
				t.Errorf("history cannot be encoded: %v", err)
			}
			if len(field.Snapshot().Robots) != 0 {
				t.Error("finished robots remained on the board")
			}
			assertClean(t, v, base)
		})
	}
}
