package robot_event_test

import (
	"fmt"
	"reflect"
	"testing"

	memory "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/memory"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
)

func TestValidPicolangEvent(t *testing.T) {
	t.Parallel()
	for _, event := range []robotevent.PicolangEvent{
		robotevent.CurrentNodeEvent{}, robotevent.VariableInitEvent{},
		robotevent.VariableUpdateEvent{}, robotevent.VariableDropEvent{},
		robotevent.ExecutionFinishedEvent{},
		robotevent.MemoryAllocateEvent{ID: 1, Value: memory.Integer{Integer: 42}}, robotevent.MemoryRetainEvent{}, robotevent.MemoryReleaseEvent{},
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

func TestInvalidAllocationsAreRejected(t *testing.T) {
	t.Parallel()
	for _, event := range []robotevent.MemoryAllocateEvent{
		{Value: memory.Unit{}},
		{ID: 1},
		{ID: 1, Value: &memory.Integer{Integer: 1}},
		{ID: 1, Value: memory.Terrain{}},
		{ID: 1, Value: memory.Rotation{Rotation: "invalid"}},
		{ID: 1, Value: memory.DistanceTarget{Target: "invalid"}},
	} {
		if robotevent.ValidPicolangEvent(event) {
			t.Errorf("accepted invalid allocation: %#v", event)
		}
	}
}
