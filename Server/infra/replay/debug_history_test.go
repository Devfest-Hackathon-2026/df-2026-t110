package replay

import (
	json "encoding/json/v2"
	"reflect"
	"strconv"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	payload "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/memory"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
	wire "github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/history"
)

func TestDebugHistoryReplaysBindingsAndReleasesMemory(t *testing.T) {
	t.Parallel()
	const hint protocol.HintID = "00000000-0000-4000-8000-000000000001"
	robot := func(id protocol.RobotID, event robotevent.PicolangEvent) gameevent.GameEvent {
		return gameevent.RobotEvent{InterpreterID: id, Event: event}
	}
	history, err := encodeHistory([]gameevent.GameEvent{
		robot("a", robotevent.MemoryAllocateEvent{ID: 1, Value: payload.Integer{Integer: 1}}),
		robot("a", robotevent.MemoryAllocateEvent{ID: 2, Value: payload.ListNode{Prev: 0, Element: 1}}),
		robot("a", robotevent.MemoryReleaseEvent{ID: 1}),
		robot("a", robotevent.VariableInitEvent{HintID: hint, ValueID: 2}),
		robot("b", robotevent.MemoryAllocateEvent{ID: 1, Value: payload.Integer{Integer: 9}}),
		robot("b", robotevent.MemoryAllocateEvent{ID: 2, Value: payload.ListNode{Prev: 0, Element: 1}}),
		robot("b", robotevent.MemoryReleaseEvent{ID: 1}),
		robot("b", robotevent.VariableInitEvent{HintID: hint, ValueID: 2}),
		robot("a", robotevent.MemoryRetainEvent{ID: 2}),
		robot("a", robotevent.VariableInitEvent{HintID: hint, ValueID: 2}),
		robot("a", robotevent.MemoryAllocateEvent{ID: 3, Value: payload.Integer{Integer: 3}}),
		robot("a", robotevent.MemoryAllocateEvent{ID: 4, Value: payload.ListNode{Prev: 2, Element: 3}}),
		robot("a", robotevent.MemoryReleaseEvent{ID: 3}),
		robot("a", robotevent.VariableUpdateEvent{HintID: hint, ValueID: 4}),
		robot("a", robotevent.MemoryReleaseEvent{ID: 2}),
		robot("a", robotevent.VariableDropEvent{HintID: hint}),
		robot("a", robotevent.MemoryReleaseEvent{ID: 4}),
		robot("a", robotevent.MemoryAllocateEvent{ID: 5, Value: payload.ListNode{Prev: 0, Element: 2}}),
		robot("a", robotevent.VariableUpdateEvent{HintID: hint, ValueID: 5}),
		robot("a", robotevent.MemoryReleaseEvent{ID: 2}),
		robot("b", robotevent.VariableDropEvent{HintID: hint}),
		robot("b", robotevent.MemoryReleaseEvent{ID: 2}),
		robot("a", robotevent.VariableDropEvent{HintID: hint}),
		robot("a", robotevent.MemoryReleaseEvent{ID: 5}),
	})
	if err != nil {
		t.Fatal(err)
	}

	stores := make(map[string]*memory.Store)
	bindings := make(map[string][]memory.ID)
	var observed []string
	for _, item := range history {
		event := item.(wire.PicolangEvent)
		store := stores[event.InterpreterId]
		if store == nil {
			store = &memory.Store{}
			stores[event.InterpreterId] = store
		}
		debug := event.Event
		parseID := func(value wire.PicolangMemoryId) memory.ID {
			n, err := strconv.ParseUint(string(value), 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			return memory.ID(n)
		}
		switch {
		case debug.MemoryAllocate != nil:
			entry := debug.MemoryAllocate
			var value memory.Value
			switch {
			case entry.Value.Integer != nil:
				value = memory.Integer{Value: int64(entry.Value.Integer.Value)}
			case entry.Value.ListNode != nil:
				node := entry.Value.ListNode
				value = memory.ListNode{Prev: parseID(node.Prev), Value: parseID(node.Value)}
			default:
				t.Fatalf("unexpected allocation %+v", entry.Value)
			}
			id, err := store.Add(value)
			if err != nil {
				t.Fatal(err)
			}
			if id != parseID(entry.Id) {
				t.Errorf("allocated ID %d, event ID %s", id, entry.Id)
			}
		case debug.MemoryRetain != nil:
			if err := store.Retain(parseID(debug.MemoryRetain.Id)); err != nil {
				t.Fatal(err)
			}
		case debug.MemoryRelease != nil:
			if err := store.Release(parseID(debug.MemoryRelease.Id)); err != nil {
				t.Fatal(err)
			}
		case debug.VariableInit != nil:
			bindings[event.InterpreterId] = append(bindings[event.InterpreterId], parseID(debug.VariableInit.ValueId))
		case debug.VariableUpdate != nil:
			stack := bindings[event.InterpreterId]
			stack[len(stack)-1] = parseID(debug.VariableUpdate.ValueId)
		case debug.VariableDrop != nil:
			stack := bindings[event.InterpreterId]
			bindings[event.InterpreterId] = stack[:len(stack)-1]
		}
		for _, id := range bindings[event.InterpreterId] {
			if _, ok := store.Get(id); !ok {
				t.Errorf("binding in %s refers to released ID %d", event.InterpreterId, id)
			}
		}
		if debug.VariableInit != nil || debug.VariableUpdate != nil || debug.VariableDrop != nil {
			stack := bindings[event.InterpreterId]
			if len(stack) > 0 {
				encoded, err := json.Marshal(replayMemoryValue(t, store, stack[len(stack)-1]))
				if err != nil {
					t.Fatal(err)
				}
				observed = append(observed, event.InterpreterId+":"+string(encoded))
			}
		}
	}
	want := []string{"a:[1]", "b:[9]", "a:[1]", "a:[1,3]", "a:[1]", "a:[[1]]"}
	if !reflect.DeepEqual(observed, want) {
		t.Errorf("replayed bindings = %v, want %v", observed, want)
	}
	for name, store := range stores {
		var empty memory.Store
		if store.MemoryUsage() != empty.MemoryUsage() {
			t.Errorf("interpreter %s retains memory after all drops", name)
		}
		if refs, _ := store.RefCount(memory.RootID); refs != 1 {
			t.Errorf("interpreter %s root references = %d, want 1", name, refs)
		}
	}
}

func replayMemoryValue(t *testing.T, store *memory.Store, id memory.ID) any {
	t.Helper()
	value, ok := store.Get(id)
	if !ok {
		t.Fatalf("unknown memory ID %d", id)
	}
	switch value := value.(type) {
	case memory.Integer:
		return value.Value
	case memory.ListRoot:
		return []any{}
	case memory.ListNode:
		return append(replayMemoryValue(t, store, value.Prev).([]any), replayMemoryValue(t, store, value.Value))
	default:
		t.Fatalf("unexpected memory value %T", value)
		return nil
	}
}
