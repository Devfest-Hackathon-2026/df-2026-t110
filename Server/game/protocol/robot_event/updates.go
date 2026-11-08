package robot_event

import (
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

// ValueUpdate follows the primitive/compound hierarchy in the history schema.
type ValueUpdate interface{ valueUpdate() }

type PrimitiveUpdate interface {
	ValueUpdate
	primitiveUpdate()
}

type CompoundUpdate interface {
	ValueUpdate
	compoundUpdate()
}

type BoolUpdate struct{ Value bool }
type IntegerUpdate struct{ Value int64 }
type UnitUpdate struct{}
type TerrainUpdate struct{ Value values.TerrainValue }
type DistanceTargetUpdate struct{ Value values.DistanceTarget }
type RotationUpdate struct{ Value values.Rotation }
type StackUpdate struct{ Update StackOperation }

func (BoolUpdate) valueUpdate()           {}
func (IntegerUpdate) valueUpdate()        {}
func (UnitUpdate) valueUpdate()           {}
func (TerrainUpdate) valueUpdate()        {}
func (DistanceTargetUpdate) valueUpdate() {}
func (RotationUpdate) valueUpdate()       {}
func (StackUpdate) valueUpdate()          {}

func (BoolUpdate) primitiveUpdate()           {}
func (IntegerUpdate) primitiveUpdate()        {}
func (UnitUpdate) primitiveUpdate()           {}
func (TerrainUpdate) primitiveUpdate()        {}
func (DistanceTargetUpdate) primitiveUpdate() {}
func (RotationUpdate) primitiveUpdate()       {}
func (StackUpdate) compoundUpdate()           {}

// StackOperation only permits operations supported by picolang stacks.
type StackOperation interface{ stackOperation() }

type StackReplace struct{ Value []values.RecordValue }
type PushBack struct{ Value values.RecordValue }
type PopBack struct{}

func (StackReplace) stackOperation() {}
func (PushBack) stackOperation()     {}
func (PopBack) stackOperation()      {}
