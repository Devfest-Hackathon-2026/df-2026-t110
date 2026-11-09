package testutil

import (
	"slices"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	payload "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/memory"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
)

// StackValue is a test-only decoded value; production events never hold arrays.
type StackValue struct{ Elements []any }

// EventReplay reconstructs a fresh store solely from the recorded operations.
// It checks ID ordering, reference lifetimes and recursive variable bindings.
type EventReplay struct {
	Memory   memory.Store
	Bindings map[protocol.HintID][]memory.ID
}

func (r *EventReplay) Apply(t *testing.T, event robotevent.PicolangEvent) {
	t.Helper()
	if r.Bindings == nil {
		r.Bindings = make(map[protocol.HintID][]memory.ID)
	}
	switch e := event.(type) {
	case robotevent.MemoryAllocateEvent:
		id, err := r.Memory.Add(MemoryValue(t, e.Value))
		if err != nil {
			t.Fatalf("replay allocation %d: %v", e.ID, err)
		}
		if id != e.ID {
			t.Errorf("allocated replay ID %d, recorded ID %d", id, e.ID)
		}
	case robotevent.MemoryRetainEvent:
		if err := r.Memory.Retain(e.ID); err != nil {
			t.Fatalf("replay retain %d: %v", e.ID, err)
		}
	case robotevent.MemoryReleaseEvent:
		if err := r.Memory.Release(e.ID); err != nil {
			t.Fatalf("replay release %d: %v", e.ID, err)
		}
	case robotevent.VariableInitEvent:
		r.Bindings[e.HintID] = append(r.Bindings[e.HintID], e.ValueID)
	case robotevent.VariableUpdateEvent:
		bindings := r.Bindings[e.HintID]
		if len(bindings) == 0 {
			t.Fatalf("update of unknown replay variable %s", e.HintID)
		}
		bindings[len(bindings)-1] = e.ValueID
	case robotevent.VariableDropEvent:
		bindings := r.Bindings[e.HintID]
		if len(bindings) == 0 {
			t.Fatalf("drop of unknown replay variable %s", e.HintID)
		}
		r.Bindings[e.HintID] = bindings[:len(bindings)-1]
	case robotevent.ExecutionFinishedEvent:
		r.AssertEmpty(t)
	}
	for hint, bindings := range r.Bindings {
		for _, id := range bindings {
			if _, ok := r.Memory.Get(id); !ok {
				t.Errorf("replay variable %s refers to released ID %d", hint, id)
			}
		}
	}
}

func (r *EventReplay) AssertEmpty(t *testing.T) {
	t.Helper()
	var empty memory.Store
	if r.Memory.MemoryUsage() != empty.MemoryUsage() {
		t.Error("replay retains values after cleanup")
	}
	if refs, _ := r.Memory.RefCount(memory.RootID); refs != 1 {
		t.Errorf("replay root references = %d, want 1", refs)
	}
	for hint, bindings := range r.Bindings {
		if len(bindings) != 0 {
			t.Errorf("replay variable %s still has %d bindings", hint, len(bindings))
		}
	}
}

func (r *EventReplay) Value(t *testing.T, id memory.ID) any {
	t.Helper()
	value, ok := r.Memory.Get(id)
	if !ok {
		t.Fatalf("unknown replay ID %d", id)
	}
	switch v := value.(type) {
	case memory.Bool:
		return values.BoolValue{Bool: v.Value}
	case memory.Integer:
		return values.IntegerValue{Integer: v.Value}
	case memory.Unit:
		return values.UnitValue{}
	case memory.Terrain:
		return v.Value
	case memory.Rotation:
		return v.Value
	case memory.DistanceTarget:
		return v.Value
	case memory.ListRoot:
		return StackValue{}
	case memory.ListNode:
		var elements []any
		for {
			elements = append(elements, r.Value(t, v.Value))
			prev, ok := r.Memory.Get(v.Prev)
			if !ok {
				t.Fatalf("list node refers to unknown ID %d", v.Prev)
			}
			switch prev := prev.(type) {
			case memory.ListRoot:
				slices.Reverse(elements)
				return StackValue{Elements: elements}
			case memory.ListNode:
				v = prev
			default:
				t.Fatalf("list previous ID %d has value %T", v.Prev, prev)
			}
		}
	default:
		t.Fatalf("unsupported replay value %T", value)
		return nil
	}
}

// MemoryValue decodes one allocation payload without resolving child IDs.
func MemoryValue(t *testing.T, value payload.Value) memory.Value {
	t.Helper()
	switch v := value.(type) {
	case payload.Bool:
		return memory.Bool{Value: v.Bool}
	case payload.Integer:
		return memory.Integer{Value: v.Integer}
	case payload.Unit:
		return memory.Unit{}
	case payload.Terrain:
		return memory.Terrain{Value: v.Terrain}
	case payload.Rotation:
		return memory.Rotation{Value: v.Rotation}
	case payload.DistanceTarget:
		return memory.DistanceTarget{Value: v.Target}
	case payload.ListNode:
		return memory.ListNode{Prev: v.Prev, Value: v.Element}
	default:
		t.Fatalf("unexpected allocation %T", value)
		return nil
	}
}
