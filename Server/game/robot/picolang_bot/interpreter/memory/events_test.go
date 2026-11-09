package memory_test

import (
	"reflect"
	"testing"

	payload "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/memory"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/internal/testutil"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/event"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
)

func TestOperationsRecordOnlyExplicitReferences(t *testing.T) {
	t.Parallel()
	var events event.Collector
	store := memory.New(&events)
	base := store.MemoryUsage()
	root := add(t, store, memory.ListRoot{})
	item := add(t, store, memory.Integer{Value: 42})
	node := add(t, store, memory.ListNode{Prev: root, Value: item})
	if err := store.Retain(node); err != nil {
		t.Fatal(err)
	}
	store.Get(node)
	store.RefCount(node)
	store.MemoryUsage()
	store.PeakMemoryUsage()
	release(t, store, item)
	release(t, store, root)
	release(t, store, node)
	release(t, store, node)
	want := []robotevent.PicolangEvent{
		robotevent.MemoryRetainEvent{ID: root},
		robotevent.MemoryAllocateEvent{ID: item, Value: payload.Integer{Integer: 42}},
		robotevent.MemoryAllocateEvent{ID: node, Value: payload.ListNode{Prev: root, Element: item}},
		robotevent.MemoryRetainEvent{ID: node},
		robotevent.MemoryReleaseEvent{ID: item},
		robotevent.MemoryReleaseEvent{ID: root},
		robotevent.MemoryReleaseEvent{ID: node},
		robotevent.MemoryReleaseEvent{ID: node},
	}
	recorded := events.Collect()
	if !reflect.DeepEqual(recorded, want) {
		t.Errorf("events = %#v, want %#v", recorded, want)
	}
	var replay testutil.EventReplay
	for _, e := range recorded {
		replay.Apply(t, e)
	}
	replay.AssertEmpty(t)
	assertUsage(t, store, base)
}

func TestFailedOperationsDoNotEmitEvents(t *testing.T) {
	t.Parallel()
	var events event.Collector
	store := memory.New(&events)
	base := store.MemoryUsage()
	for _, value := range []memory.Value{nil, &memory.Integer{Value: 1}, memory.ListNode{Prev: 999, Value: 0}, memory.Rotation{Value: "invalid"}} {
		if _, err := store.Add(value); err == nil {
			t.Errorf("accepted invalid allocation %T", value)
		}
	}
	if err := store.Retain(999); err == nil {
		t.Error("retained unknown ID")
	}
	if err := store.Release(999); err == nil {
		t.Error("released unknown ID")
	}
	if err := store.Release(memory.RootID); err == nil {
		t.Error("released store-owned root")
	}
	if recorded := events.Collect(); len(recorded) != 0 {
		t.Errorf("failed operations emitted %#v", recorded)
	}
	assertUsage(t, store, base)
}

func TestSharedRootEdgesAreAppliedTwice(t *testing.T) {
	t.Parallel()
	var events event.Collector
	store := memory.New(&events)
	id := add(t, store, memory.ListNode{Prev: 0, Value: 0})
	if refs, _ := store.RefCount(0); refs != 3 {
		t.Errorf("root count = %d, want 3", refs)
	}
	release(t, store, id)
	var replay testutil.EventReplay
	for _, e := range events.Collect() {
		replay.Apply(t, e)
	}
	replay.AssertEmpty(t)
}
