package tutorialrun

import (
	json "encoding/json/v2"
	"io/fs"
	"math"
	"net/url"
	"os"
	"strings"
	"testing"

	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	payload "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/memory"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	"github.com/google/jsonschema-go/jsonschema"
	"gopkg.in/yaml.v3"
)

// Check the adapter against the shared schema rather than duplicating its
// field-by-field implementation in assertions.
func TestHistoryMatchesSchema(t *testing.T) {
	t.Parallel()
	order := int64(2)
	to := &fieldevent.Position{X: 1}
	const hint = "00000000-0000-4000-8000-000000000001"
	events := []gameevent.GameEvent{
		gameevent.FieldEvent{Event: fieldevent.FieldEvent{State: fieldevent.InitialState{
			Terrain: [][]fieldevent.Cell{{
				{Type: "land", Kind: &fieldevent.LandKind{Type: "plain"}, HasLuggage: true, TerritoryOwner: 1},
				{Type: "land", Kind: &fieldevent.LandKind{Type: "checkpoint", Order: &order}}, {Type: "rock"}, {Type: "sea"},
			}},
			Robots: []fieldevent.RobotState{{ID: "r", Direction: "east", TeamID: 1, CarryingLuggage: true}},
		}}},
		gameevent.FieldEvent{Event: fieldevent.FieldEvent{Tick: 1, State: fieldevent.UpdateState{Actions: []fieldevent.ActionResult{
			fieldevent.MoveResult{Kind: fieldevent.Forward, RobotID: "r", Success: true, To: to},
			fieldevent.MoveResult{Kind: fieldevent.Back, RobotID: "r"},
			fieldevent.RotateResult{RobotID: "r", Rotation: values.Left, Direction: "north"},
			fieldevent.DrillResult{RobotID: "r", Success: true, Target: to},
			fieldevent.PickUpResult{RobotID: "r", Success: true, Target: to},
			fieldevent.ReleaseResult{RobotID: "r", Success: true, Target: to},
			fieldevent.ReleaseResult{RobotID: "r", Success: true, Discarded: true},
			fieldevent.WaitResult{RobotID: "r", Ticks: 2},
			fieldevent.PrintIntegerResult{RobotID: "r", Integer: math.MaxInt64},
			fieldevent.PrintBoolResult{RobotID: "r"}, fieldevent.PrintUnitResult{RobotID: "r"},
			fieldevent.PrintTerrainResult{RobotID: "r", Terrain: values.LandTerrain{Kind: values.PlainLand{}, Territory: values.OwnTerritory}},
			fieldevent.PrintRotationResult{RobotID: "r", Rotation: values.Right},
			fieldevent.PrintDistanceTargetResult{RobotID: "r", DistanceTarget: values.DistanceGoal},
		}}}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryAllocateEvent{ID: 1, Value: payload.Bool{}}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryAllocateEvent{ID: 2, Value: payload.Integer{Integer: math.MaxInt64}}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryAllocateEvent{ID: 3, Value: payload.Unit{}}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryAllocateEvent{ID: 4, Value: payload.Rotation{Rotation: values.Left}}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryAllocateEvent{ID: 5, Value: payload.DistanceTarget{Target: values.DistanceGoal}}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryAllocateEvent{ID: 6, Value: payload.Terrain{Terrain: values.LandTerrain{Kind: values.GoalLand{}, HasLuggage: true}}}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryAllocateEvent{ID: 7, Value: payload.Terrain{Terrain: values.LandTerrain{Kind: values.CheckpointLand{Order: 2}}}}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryAllocateEvent{ID: 8, Value: payload.Terrain{Terrain: values.RockTerrain{}}}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryAllocateEvent{ID: 9, Value: payload.Terrain{Terrain: values.SeaTerrain{}}}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryAllocateEvent{ID: 10, Value: payload.ListNode{Prev: 0, Element: 2}}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryAllocateEvent{ID: 11, Value: payload.ListNode{Prev: 0, Element: 10}}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.VariableInitEvent{HintID: hint, ValueID: 10}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.VariableUpdateEvent{HintID: hint, ValueID: 11}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryReleaseEvent{ID: 10}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryRetainEvent{ID: 0}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryReleaseEvent{ID: 0}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.CurrentNodeEvent{HintID: hint}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.VariableDropEvent{HintID: hint}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryReleaseEvent{ID: 11}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryReleaseEvent{ID: 1}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryReleaseEvent{ID: 2}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryReleaseEvent{ID: 3}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryReleaseEvent{ID: 4}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryReleaseEvent{ID: 5}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryReleaseEvent{ID: 6}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryReleaseEvent{ID: 7}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryReleaseEvent{ID: 8}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.MemoryReleaseEvent{ID: 9}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.ExecutionFinishedEvent{HintID: hint, Reason: robotevent.EndCompleted, Message: "program completed"}},
	}
	history, err := encodeHistory(events)
	if err != nil {
		t.Fatalf("encode history: %v", err)
	}
	raw, err := json.Marshal(history)
	if err != nil {
		t.Errorf("marshal history: %v", err)
		return
	}
	if !strings.Contains(string(raw), "9223372036854775807") {
		t.Error("history lost i64 precision")
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Errorf("decode history: %v", err)
		return
	}
	loader := func(uri *url.URL) (*jsonschema.Schema, error) {
		raw, err := fs.ReadFile(os.DirFS("../../../Schema"), strings.TrimPrefix(uri.Path, "/"))
		if err != nil {
			return nil, err
		}
		var doc any
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			return nil, err
		}
		encoded, err := json.Marshal(doc)
		if err != nil {
			return nil, err
		}
		var schema jsonschema.Schema
		if err := json.Unmarshal(encoded, &schema); err != nil {
			return nil, err
		}
		return &schema, nil
	}
	root, err := url.Parse("https://schemas.local/game_event/history.schema.yaml")
	if err != nil {
		t.Fatal(err)
	}
	schema, err := loader(root)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := schema.Resolve(&jsonschema.ResolveOptions{BaseURI: root.String(), Loader: loader})
	if err != nil {
		t.Fatal(err)
	}
	if err := resolved.Validate(value); err != nil {
		t.Errorf("history violates shared schema: %v", err)
	}
}
