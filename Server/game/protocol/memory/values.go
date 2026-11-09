// Package memory defines the immutable payloads of recorded memory operations.
package memory

import values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"

// ID is local to one interpreter. Transport encodes its uint64 as decimal text.
type ID uint64

const RootID ID = 0

// Value is a primitive allocation, list node or option. The common root is implicit.
type Value interface{ memoryValue() }

type Bool struct {
	Bool bool
	Value
}
type Integer struct {
	Integer int64
	Value
}
type Unit struct{ Value }
type Terrain struct {
	Terrain values.TerrainValue
	Value
}
type Rotation struct {
	Rotation values.Rotation
	Value
}
type DistanceTarget struct {
	Target values.DistanceTarget
	Value
}

// ListNode shares the previous stack and the final element by ID. Element can
// itself be a stack. Allocating a node retains both edges exactly once.
type ListNode struct {
	Prev    ID
	Element ID
	Value
}

// Some retains a single existing value by ID. None has no outgoing edge.
type Some struct {
	Element ID
	Value
}
type None struct{ Value }
