package robot_event

import memory "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/memory"

// MemoryAllocateEvent creates one owned reference. List nodes also retain both
// outgoing edges; those implicit retains do not have their own events.
type MemoryAllocateEvent struct {
	ID    memory.ID
	Value memory.Value
	PicolangEvent
}

// MemoryRetainEvent acquires one owned reference, including to the empty root.
type MemoryRetainEvent struct {
	ID memory.ID
	PicolangEvent
}

// MemoryReleaseEvent releases one owned reference. Deleting nodes recursively
// releases their edges without emitting additional release events.
type MemoryReleaseEvent struct {
	ID memory.ID
	PicolangEvent
}
