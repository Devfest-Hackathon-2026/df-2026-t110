package robot_event_test

import (
	"encoding/json"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

const hint = protocol.HintID("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")

func TestRecordEncoding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		message robotevent.PicolangEvent
		want    string
	}{
		{"current node", robotevent.CurrentNodeEvent{HintID: hint}, `{"kind":"current_node","hintId":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}`},
		{"drop", robotevent.VariableDropEvent{HintID: hint}, `{"kind":"variable_drop","hintId":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			encoded, err := json.Marshal(tt.message)
			if err != nil || string(encoded) != tt.want {
				t.Errorf("record = %s, %v; want %s", encoded, err, tt.want)
			}
		})
	}
}

func TestInitialValueEncoding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value values.RecordValue
		want  string
	}{
		{"bool", values.BoolValue{Bool: false}, "false"},
		{"integer", values.IntegerValue{Integer: -9223372036854775808}, "-9223372036854775808"},
		{"unit", values.UnitValue{}, "null"},
		{"distance target", values.DistanceGoalArea, `"goal_area"`},
		{"rotation", values.RotationRecordValue{Rotation: values.Left}, `"left"`},
		{"plain", values.LandTerrain{Kind: values.PlainLand{}}, `{"type":"land","kind":{"type":"plain"}}`},
		{"goal", values.LandTerrain{Kind: values.GoalLand{}}, `{"type":"land","kind":{"type":"goal"}}`},
		{"checkpoint", values.LandTerrain{Kind: values.CheckpointLand{Order: 0}}, `{"type":"land","kind":{"type":"checkpoint","order":0}}`},
		{"luggage", values.LandTerrain{Kind: values.PlainLand{}, HasLuggage: true, GoalArea: values.OwnGoalArea}, `{"type":"land","kind":{"type":"plain"},"hasLuggage":true,"goalArea":"own"}`},
		{"rock", values.RockTerrain{}, `{"type":"rock"}`},
		{"sea", values.SeaTerrain{}, `{"type":"sea"}`},
		{"empty vector", values.VectorRecordValue{}, "[]"},
		{"nested vector", values.VectorRecordValue{Elements: []values.RecordValue{
			values.VectorRecordValue{Elements: []values.RecordValue{values.IntegerValue{Integer: 3}}},
			values.VectorRecordValue{},
		}}, "[[3],[]]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			encoded, err := json.Marshal(robotevent.VariableInitEvent{HintID: hint, Value: tt.value})
			want := `{"kind":"variable_init","hintId":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","value":` + tt.want + `}`
			if err != nil || string(encoded) != want {
				t.Errorf("init = %s, %v; want %s", encoded, err, want)
			}
		})
	}
}

func TestUpdateEncoding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		diff []robotevent.ValueUpdate
		want string
	}{
		{"bool", []robotevent.ValueUpdate{robotevent.BoolUpdate{Value: true}}, `[{"type":"bool","value":true}]`},
		{"integer", []robotevent.ValueUpdate{robotevent.IntegerUpdate{Value: 5}}, `[{"type":"integer","value":5}]`},
		{"unit", []robotevent.ValueUpdate{robotevent.UnitUpdate{}}, `[{"type":"unit","value":null}]`},
		{"terrain", []robotevent.ValueUpdate{robotevent.TerrainUpdate{Value: values.LandTerrain{Kind: values.GoalLand{}}}}, `[{"type":"terrain","value":{"type":"land","kind":{"type":"goal"}}}]`},
		{"distance target", []robotevent.ValueUpdate{robotevent.DistanceTargetUpdate{Value: values.DistanceSea}}, `[{"type":"distance_target","value":"sea"}]`},
		{"rotation", []robotevent.ValueUpdate{robotevent.RotationUpdate{Value: values.Right}}, `[{"type":"rotation","value":"right"}]`},
		{"vector operations", []robotevent.ValueUpdate{
			robotevent.VectorUpdate{Update: robotevent.PushBack{Value: values.IntegerValue{Integer: 2}}},
			robotevent.VectorUpdate{Update: robotevent.PopBack{}},
			robotevent.VectorUpdate{Update: robotevent.VectorReplace{}},
		}, `[{"type":"vector","update":{"op":"push_back","value":2}},{"type":"vector","update":{"op":"pop_back"}},{"type":"vector","update":{"op":"replace","value":[]}}]`},
		{"nested vector operand", []robotevent.ValueUpdate{
			robotevent.VectorUpdate{Update: robotevent.PushBack{Value: values.VectorRecordValue{Elements: []values.RecordValue{values.IntegerValue{Integer: 3}}}}},
		}, `[{"type":"vector","update":{"op":"push_back","value":[3]}}]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			encoded, err := json.Marshal(robotevent.VariableUpdateEvent{HintID: hint, Diff: tt.diff})
			want := `{"kind":"variable_update","hintId":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","diff":` + tt.want + `}`
			if err != nil || string(encoded) != want {
				t.Errorf("update = %s, %v; want %s", encoded, err, want)
			}
		})
	}
}

func TestInvalidRecordPayloads(t *testing.T) {
	t.Parallel()
	for name, value := range map[string]values.RecordValue{
		"nil":               nil,
		"typed nil":         (*values.BoolValue)(nil),
		"pointer":           &values.BoolValue{Bool: true},
		"rotation":          values.RotationRecordValue{Rotation: "sideways"},
		"land kind":         values.LandTerrain{},
		"land kind pointer": values.LandTerrain{Kind: &values.PlainLand{}},
	} {
		t.Run("init/"+name, func(t *testing.T) {
			t.Parallel()
			if _, err := json.Marshal(robotevent.VariableInitEvent{HintID: hint, Value: value}); err == nil {
				t.Error("invalid initial value accepted")
			}
		})
	}
	for name, update := range map[string]robotevent.ValueUpdate{
		"pointer":                  &robotevent.IntegerUpdate{Value: 1},
		"nil terrain":              robotevent.TerrainUpdate{},
		"rotation":                 robotevent.RotationUpdate{Value: "sideways"},
		"pointer vector operation": robotevent.VectorUpdate{Update: &robotevent.PopBack{}},
		"missing push value":       robotevent.VectorUpdate{Update: robotevent.PushBack{}},
	} {
		t.Run("update/"+name, func(t *testing.T) {
			t.Parallel()
			if _, err := json.Marshal(robotevent.VariableUpdateEvent{HintID: hint, Diff: []robotevent.ValueUpdate{update}}); err == nil {
				t.Error("invalid update accepted")
			}
		})
	}
}
