package tutorialrun

import (
	"fmt"
	"strconv"

	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
	wire "github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/history"
)

// debugHistory bridges the interpreter's existing snapshot events to the memory
// wire protocol until the interpreter emits memory operations directly. Each
// interpreter has its own store and a binding stack for recursive declarations.
type debugHistory struct {
	store     memory.Store
	variables map[protocol.HintID][]memory.ID
	events    []wire.PicolangDebugEvent
	err       error
}

func memoryID(id memory.ID) wire.PicolangMemoryId {
	return wire.PicolangMemoryId(strconv.FormatUint(uint64(id), 10))
}

func (h *debugHistory) encode(event robotevent.PicolangEvent) ([]wire.PicolangDebugEvent, error) {
	h.events = nil
	switch e := event.(type) {
	case robotevent.CurrentNodeEvent:
		h.events = append(h.events, wire.PicolangDebugEvent{CurrentNode: &wire.PicolangCurrentNodeEvent{Kind: "current_node", HintId: wire.HintId(e.HintID)}})
	case robotevent.VariableInitEvent:
		id := h.record(e.Value)
		h.variables[e.HintID] = append(h.variables[e.HintID], id)
		h.events = append(h.events, wire.PicolangDebugEvent{VariableInit: &wire.PicolangVariableInitEvent{Kind: "variable_init", HintId: wire.HintId(e.HintID), ValueId: memoryID(id)}})
	case robotevent.VariableUpdateEvent:
		bindings := h.variables[e.HintID]
		if len(bindings) == 0 || len(e.Diff) == 0 {
			return nil, fmt.Errorf("invalid update for variable %s", e.HintID)
		}
		for _, update := range e.Diff {
			previous := bindings[len(bindings)-1]
			id := h.update(previous, update)
			bindings[len(bindings)-1] = id
			h.events = append(h.events, wire.PicolangDebugEvent{VariableUpdate: &wire.PicolangVariableUpdateEvent{Kind: "variable_update", HintId: wire.HintId(e.HintID), ValueId: memoryID(id)}})
			h.release(previous)
		}
	case robotevent.VariableDropEvent:
		bindings := h.variables[e.HintID]
		if len(bindings) == 0 {
			return nil, fmt.Errorf("drop of unknown variable %s", e.HintID)
		}
		h.variables[e.HintID] = bindings[:len(bindings)-1]
		h.events = append(h.events, wire.PicolangDebugEvent{VariableDrop: &wire.PicolangVariableDropEvent{Kind: "variable_drop", HintId: wire.HintId(e.HintID)}})
		h.release(bindings[len(bindings)-1])
	case robotevent.ExecutionFinishedEvent:
		h.events = append(h.events, wire.PicolangDebugEvent{ExecutionFinished: &wire.PicolangExecutionFinishedEvent{Kind: "execution_finished", HintId: wire.HintId(e.HintID), Reason: wire.PicolangExecutionFinishedEventReason(e.Reason), Message: e.Message}})
	default:
		return nil, fmt.Errorf("unsupported debug event %T", event)
	}
	return h.events, h.err
}

func (h *debugHistory) allocate(value memory.Value) memory.ID {
	if h.err != nil {
		return 0
	}
	payload, err := memoryValue(value)
	if err != nil {
		h.err = err
		return 0
	}
	id, err := h.store.Add(value)
	if err != nil {
		h.err = err
		return 0
	}
	h.events = append(h.events, wire.PicolangDebugEvent{MemoryAllocate: &wire.PicolangMemoryAllocateEvent{Kind: "memory_allocate", Id: memoryID(id), Value: payload}})
	return id
}

func (h *debugHistory) retain(id memory.ID) memory.ID {
	if h.err == nil {
		h.err = h.store.Retain(id)
		if h.err == nil {
			h.events = append(h.events, wire.PicolangDebugEvent{MemoryRetain: &wire.PicolangMemoryRetainEvent{Kind: "memory_retain", Id: memoryID(id)}})
		}
	}
	return id
}

func (h *debugHistory) release(id memory.ID) {
	if h.err == nil {
		h.err = h.store.Release(id)
		if h.err == nil {
			h.events = append(h.events, wire.PicolangDebugEvent{MemoryRelease: &wire.PicolangMemoryReleaseEvent{Kind: "memory_release", Id: memoryID(id)}})
		}
	}
}

func (h *debugHistory) record(value values.RecordValue) memory.ID {
	switch v := value.(type) {
	case values.BoolValue:
		return h.allocate(memory.Bool{Value: v.Bool})
	case values.IntegerValue:
		return h.allocate(memory.Integer{Value: v.Integer})
	case values.UnitValue:
		return h.allocate(memory.Unit{})
	case values.RotationRecordValue:
		return h.allocate(memory.Rotation{Value: v.Rotation})
	case values.DistanceTarget:
		return h.allocate(memory.DistanceTarget{Value: v})
	case values.TerrainValue:
		return h.allocate(memory.Terrain{Value: v})
	case values.StackRecordValue:
		id := h.retain(memory.RootID)
		for _, element := range v.Elements {
			child := h.record(element)
			next := h.allocate(memory.ListNode{Prev: id, Value: child})
			h.release(child)
			h.release(id)
			id = next
		}
		return id
	default:
		h.err = fmt.Errorf("unsupported recorded value %T", value)
		return 0
	}
}

func (h *debugHistory) update(previous memory.ID, update robotevent.ValueUpdate) memory.ID {
	switch u := update.(type) {
	case robotevent.BoolUpdate:
		return h.allocate(memory.Bool{Value: u.Value})
	case robotevent.IntegerUpdate:
		return h.allocate(memory.Integer{Value: u.Value})
	case robotevent.UnitUpdate:
		return h.allocate(memory.Unit{})
	case robotevent.TerrainUpdate:
		return h.allocate(memory.Terrain{Value: u.Value})
	case robotevent.RotationUpdate:
		return h.allocate(memory.Rotation{Value: u.Value})
	case robotevent.DistanceTargetUpdate:
		return h.allocate(memory.DistanceTarget{Value: u.Value})
	case robotevent.StackUpdate:
		switch op := u.Update.(type) {
		case robotevent.StackReplace:
			return h.record(values.StackRecordValue{Elements: op.Value})
		case robotevent.PushBack:
			child := h.record(op.Value)
			id := h.allocate(memory.ListNode{Prev: previous, Value: child})
			h.release(child)
			return id
		case robotevent.PopBack:
			value, _ := h.store.Get(previous)
			if node, ok := value.(memory.ListNode); ok {
				return h.retain(node.Prev)
			}
		}
	}
	h.err = fmt.Errorf("invalid value update %T for memory ID %d", update, previous)
	return 0
}

func memoryValue(value memory.Value) (wire.PicolangMemoryValue, error) {
	switch v := value.(type) {
	case memory.Bool:
		return wire.PicolangMemoryValue{Bool: &wire.BoolValue{Type: "bool", Value: v.Value}}, nil
	case memory.Integer:
		return wire.PicolangMemoryValue{Integer: &wire.IntegerValue{Type: "integer", Value: int(v.Value)}}, nil
	case memory.Unit:
		return wire.PicolangMemoryValue{Unit: &wire.UnitValue{Type: "unit"}}, nil
	case memory.Terrain:
		terrain, err := terrainValue(v.Value)
		return wire.PicolangMemoryValue{Terrain: &wire.TerrainValue{Type: "terrain", Value: terrain}}, err
	case memory.Rotation:
		return wire.PicolangMemoryValue{Rotation: &wire.RotationValue{Type: "rotation", Value: wire.RotationValueValue(v.Value)}}, nil
	case memory.DistanceTarget:
		return wire.PicolangMemoryValue{DistanceTarget: &wire.DistanceTargetValue{Type: "distance_target", Value: wire.DistanceTarget(v.Value)}}, nil
	case memory.ListNode:
		return wire.PicolangMemoryValue{ListNode: &wire.ListNode{Type: "list_node", Prev: memoryID(v.Prev), Value: memoryID(v.Value)}}, nil
	default:
		return wire.PicolangMemoryValue{}, fmt.Errorf("unsupported memory value %T", value)
	}
}
