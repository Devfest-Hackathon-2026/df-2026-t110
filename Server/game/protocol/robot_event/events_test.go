package robot_event_test

import (
	"fmt"
	"reflect"
	"testing"

	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
)

func TestValidPicolangEvent(t *testing.T) {
	t.Parallel()
	for _, event := range []robotevent.PicolangEvent{
		robotevent.CurrentNodeEvent{}, robotevent.VariableInitEvent{},
		robotevent.VariableUpdateEvent{}, robotevent.VariableDropEvent{},
		robotevent.ExecutionFinishedEvent{},
	} {
		t.Run(fmt.Sprintf("%T", event), func(t *testing.T) {
			t.Parallel()
			if !robotevent.ValidPicolangEvent(event) {
				t.Errorf("value event rejected: %T", event)
			}
			pointer := reflect.New(reflect.TypeOf(event)).Interface().(robotevent.PicolangEvent)
			if robotevent.ValidPicolangEvent(pointer) {
				t.Errorf("pointer event accepted: %T", pointer)
			}
		})
	}
	if robotevent.ValidPicolangEvent(nil) || robotevent.ValidPicolangEvent((*robotevent.CurrentNodeEvent)(nil)) {
		t.Error("nil event accepted")
	}
}
