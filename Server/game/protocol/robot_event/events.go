// Package robot_event defines a single interpreter's debug events.
package robot_event

import (
	memory "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/memory"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

// PicolangEvent reports an interpreter event without a clock or interpreter ID.
type PicolangEvent interface {
	picolangEvent()
}

// ValidPicolangEvent rejects nil and pointer variants at recording boundaries.
func ValidPicolangEvent(message PicolangEvent) bool {
	switch event := message.(type) {
	case MemoryAllocateEvent:
		if event.ID == memory.RootID {
			return false
		}
		switch value := event.Value.(type) {
		case memory.Bool, memory.Integer, memory.Unit, memory.ListNode, memory.Some, memory.None:
			return true
		case memory.Terrain:
			return values.ValidValue(value.Terrain)
		case memory.Rotation:
			return value.Rotation == values.Left || value.Rotation == values.Right
		case memory.DistanceTarget:
			return value.Target.Valid()
		default:
			return false
		}
	case MemoryRetainEvent, MemoryReleaseEvent, CurrentNodeEvent, VariableInitEvent, VariableUpdateEvent, VariableDropEvent, ExecutionFinishedEvent:
		return true
	default:
		return false
	}
}
