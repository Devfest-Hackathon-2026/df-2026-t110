package game_event_test

import (
	"encoding/json"
	"testing"

	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
)

func TestGameEventVariants(t *testing.T) {
	t.Parallel()
	history := []gameevent.GameEvent{
		gameevent.FieldEvent{Event: fieldevent.FieldEvent{Tick: 7}},
		gameevent.RobotEvent{
			InterpreterID: "robot-a",
			Event:         robotevent.CurrentNodeEvent{HintID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"},
		},
	}
	want := `[{"type":"field","tick":7,"state":{}},{"type":"picolang","interpreterId":"robot-a","event":{"kind":"current_node","hintId":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}}]`
	encoded, err := json.Marshal(history)
	if err != nil || string(encoded) != want {
		t.Errorf("history = %s, %v; want %s", encoded, err, want)
	}
}
