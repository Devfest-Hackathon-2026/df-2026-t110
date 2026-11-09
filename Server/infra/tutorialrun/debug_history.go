package tutorialrun

import (
	"fmt"
	"strconv"

	payload "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/memory"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	wire "github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/history"
)

func memoryID(id payload.ID) wire.PicolangMemoryId {
	return wire.PicolangMemoryId(strconv.FormatUint(uint64(id), 10))
}

func debugEvent(event robotevent.PicolangEvent) (wire.PicolangDebugEvent, error) {
	switch e := event.(type) {
	case robotevent.CurrentNodeEvent:
		return wire.PicolangDebugEvent{CurrentNode: &wire.PicolangCurrentNodeEvent{Kind: "current_node", HintId: wire.HintId(e.HintID)}}, nil
	case robotevent.MemoryAllocateEvent:
		value, err := memoryValue(e.Value)
		return wire.PicolangDebugEvent{MemoryAllocate: &wire.PicolangMemoryAllocateEvent{Kind: "memory_allocate", Id: memoryID(e.ID), Value: value}}, err
	case robotevent.MemoryRetainEvent:
		return wire.PicolangDebugEvent{MemoryRetain: &wire.PicolangMemoryRetainEvent{Kind: "memory_retain", Id: memoryID(e.ID)}}, nil
	case robotevent.MemoryReleaseEvent:
		return wire.PicolangDebugEvent{MemoryRelease: &wire.PicolangMemoryReleaseEvent{Kind: "memory_release", Id: memoryID(e.ID)}}, nil
	case robotevent.VariableInitEvent:
		return wire.PicolangDebugEvent{VariableInit: &wire.PicolangVariableInitEvent{Kind: "variable_init", HintId: wire.HintId(e.HintID), ValueId: memoryID(e.ValueID)}}, nil
	case robotevent.VariableUpdateEvent:
		return wire.PicolangDebugEvent{VariableUpdate: &wire.PicolangVariableUpdateEvent{Kind: "variable_update", HintId: wire.HintId(e.HintID), ValueId: memoryID(e.ValueID)}}, nil
	case robotevent.VariableDropEvent:
		return wire.PicolangDebugEvent{VariableDrop: &wire.PicolangVariableDropEvent{Kind: "variable_drop", HintId: wire.HintId(e.HintID)}}, nil
	case robotevent.ExecutionFinishedEvent:
		return wire.PicolangDebugEvent{ExecutionFinished: &wire.PicolangExecutionFinishedEvent{Kind: "execution_finished", HintId: wire.HintId(e.HintID), Reason: wire.PicolangExecutionFinishedEventReason(e.Reason), Message: e.Message}}, nil
	default:
		return wire.PicolangDebugEvent{}, fmt.Errorf("unsupported debug event %T", event)
	}
}

func memoryValue(value payload.Value) (wire.PicolangMemoryValue, error) {
	switch v := value.(type) {
	case payload.Bool:
		return wire.PicolangMemoryValue{Bool: &wire.BoolValue{Type: "bool", Value: v.Bool}}, nil
	case payload.Integer:
		return wire.PicolangMemoryValue{Integer: &wire.IntegerValue{Type: "integer", Value: int(v.Integer)}}, nil
	case payload.Unit:
		return wire.PicolangMemoryValue{Unit: &wire.UnitValue{Type: "unit"}}, nil
	case payload.Terrain:
		terrain, err := terrainValue(v.Terrain)
		return wire.PicolangMemoryValue{Terrain: &wire.TerrainValue{Type: "terrain", Value: terrain}}, err
	case payload.Rotation:
		return wire.PicolangMemoryValue{Rotation: &wire.RotationValue{Type: "rotation", Value: wire.RotationValueValue(v.Rotation)}}, nil
	case payload.DistanceTarget:
		return wire.PicolangMemoryValue{DistanceTarget: &wire.DistanceTargetValue{Type: "distance_target", Value: wire.DistanceTarget(v.Target)}}, nil
	case payload.ListNode:
		return wire.PicolangMemoryValue{ListNode: &wire.ListNode{Type: "list_node", Prev: memoryID(v.Prev), Value: memoryID(v.Element)}}, nil
	case payload.Some:
		return wire.PicolangMemoryValue{Some: &wire.SomeValue{Type: "some", Value: memoryID(v.Element)}}, nil
	case payload.None:
		return wire.PicolangMemoryValue{None: &wire.NoneValue{Type: "none"}}, nil
	default:
		return wire.PicolangMemoryValue{}, fmt.Errorf("unsupported memory value %T", value)
	}
}
