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
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.VariableInitEvent{HintID: hint, Value: values.StackRecordValue{Elements: []values.RecordValue{
			values.BoolValue{}, values.IntegerValue{Integer: math.MaxInt64}, values.UnitValue{}, values.RotationRecordValue{Rotation: values.Left}, values.DistanceGoal,
			values.LandTerrain{Kind: values.GoalLand{}, HasLuggage: true}, values.LandTerrain{Kind: values.CheckpointLand{Order: 2}},
			values.RockTerrain{}, values.SeaTerrain{}, values.StackRecordValue{},
		}}}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.VariableUpdateEvent{HintID: hint, Diff: []robotevent.ValueUpdate{
			robotevent.BoolUpdate{}, robotevent.IntegerUpdate{Value: math.MaxInt64}, robotevent.UnitUpdate{},
			robotevent.TerrainUpdate{Value: values.SeaTerrain{}}, robotevent.RotationUpdate{Value: values.Right}, robotevent.DistanceTargetUpdate{Value: values.DistanceSea},
			robotevent.StackUpdate{Update: robotevent.StackReplace{}}, robotevent.StackUpdate{Update: robotevent.PushBack{Value: values.UnitValue{}}}, robotevent.StackUpdate{Update: robotevent.PopBack{}},
		}}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.CurrentNodeEvent{HintID: hint}},
		gameevent.RobotEvent{InterpreterID: "r", Event: robotevent.VariableDropEvent{HintID: hint}},
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
