// Package memory stores immutable interpreter values under reference-counted IDs.
package memory

import (
	payload "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/memory"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

// ID identifies a value within one Store. IDs must not be used in other stores.
type ID = payload.ID

// RootID identifies the common empty list, regardless of its element type.
// Store owns one reference; Add(ListRoot{}) acquires a caller-owned reference.
const RootID = payload.RootID

// Value is a stored primitive, list node or option. Add accepts only the non-pointer
// variants below, so inputs and values returned by Get cannot mutate storage.
type Value interface{ memoryValue() }

type Bool struct{ Value bool }
type Integer struct{ Value int64 }
type Unit struct{}
type Terrain struct{ Value values.TerrainValue }
type Rotation struct{ Value values.Rotation }
type DistanceTarget struct{ Value values.DistanceTarget }

// Some points to an immutable stored value by ID and retains it once.
// None has no outgoing reference; Some{Element: RootID} is some(empty stack).
type Some struct {
	Element ID
	Value
}
type None struct{ Value }

type ListRoot struct{}

// ListNode is a non-root list node. Prev must identify a list root or node;
// Value can identify any stored value, including another list.
type ListNode struct {
	Prev  ID
	Value ID
}

func (Bool) memoryValue()           {}
func (Integer) memoryValue()        {}
func (Unit) memoryValue()           {}
func (Terrain) memoryValue()        {}
func (Rotation) memoryValue()       {}
func (DistanceTarget) memoryValue() {}
func (ListRoot) memoryValue()       {}
func (ListNode) memoryValue()       {}
