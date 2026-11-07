package game_event_test

import (
	"encoding/json"
	"testing"

	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func TestGameEventVariants(t *testing.T) {
	t.Parallel()
	checkpointOrder := int64(0)
	history := []gameevent.GameEvent{
		gameevent.FieldEvent{Event: fieldevent.FieldEvent{State: fieldevent.InitialState{
			Terrain: [][]fieldevent.Cell{{{Type: "land", Kind: &fieldevent.LandKind{Type: "checkpoint", Order: &checkpointOrder}}}},
			Robots:  []fieldevent.RobotState{{ID: "robot-a", Direction: "north", DrillDurability: 3}},
		}}},
		gameevent.RobotEvent{
			InterpreterID: "robot-a",
			Event:         robotevent.CurrentNodeEvent{HintID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"},
		},
		gameevent.FieldEvent{Event: fieldevent.FieldEvent{Tick: 1, State: fieldevent.UpdateState{
			Actions: []fieldevent.ActionResult{fieldevent.MoveResult{Kind: fieldevent.Back, RobotID: "robot-a"}},
		}}},
		gameevent.FieldEvent{Event: fieldevent.FieldEvent{Tick: 2, State: fieldevent.UpdateState{
			Actions: []fieldevent.ActionResult{fieldevent.RotateResult{RobotID: "robot-a", Rotation: values.Left, Direction: "west"}},
		}}},
	}
	want := `[{"type":"field","tick":0,"state":{"kind":"initial","terrain":[[{"type":"land","kind":{"type":"checkpoint","order":0}}]],` +
		`"robots":[{"id":"robot-a","position":{"x":0,"y":0},"direction":"north","drillDurability":3}]}},` +
		`{"type":"picolang","interpreterId":"robot-a","event":{"kind":"current_node","hintId":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}},` +
		`{"type":"field","tick":1,"state":{"kind":"update","actions":[{"kind":"back","robotId":"robot-a","success":false}]}},` +
		`{"type":"field","tick":2,"state":{"kind":"update","actions":[{"kind":"rotate","robotId":"robot-a","rotation":"left","direction":"west"}]}}]`
	encoded, err := json.Marshal(history)
	if err != nil || string(encoded) != want {
		t.Errorf("history = %s, %v; want %s", encoded, err, want)
	}
}
