package memory_test

import (
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/internal/testutil"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/event"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
)

func TestOptionReferencesAndReplay(t *testing.T) {
	t.Parallel()
	var events event.Collector
	s := memory.New(&events)
	base := s.MemoryUsage()
	item := add(t, s, memory.Integer{Value: 42})
	some := add(t, s, memory.Some{Element: item})
	other := add(t, s, memory.Some{Element: item})
	none := add(t, s, memory.None{})
	nested := add(t, s, memory.Some{Element: none})
	list := add(t, s, memory.ListNode{Prev: memory.RootID, Value: some})
	outer := add(t, s, memory.Some{Element: list})
	assertCount(t, s, item, 3)
	assertCount(t, s, some, 2)
	assertCount(t, s, none, 2)
	assertCount(t, s, list, 2)
	if err := s.Retain(outer); err != nil {
		t.Fatal(err)
	}
	assertCount(t, s, outer, 2)
	assertCount(t, s, list, 2) // Copying an option shares its immutable payload.
	for _, id := range []memory.ID{item, some, none, list, outer} {
		release(t, s, id)
	}
	assertValue(t, s, item, memory.Integer{Value: 42})
	assertValue(t, s, some, memory.Some{Element: item})
	assertValue(t, s, none, memory.None{})
	release(t, s, outer)
	assertFreed(t, s, some, list, outer)
	assertCount(t, s, item, 1)
	release(t, s, other)
	release(t, s, nested)
	assertFreed(t, s, item, other, none, nested)
	assertUsage(t, s, base)
	var replay testutil.EventReplay
	for _, e := range events.Collect() {
		replay.Apply(t, e)
	}
	replay.AssertEmpty(t)
}

func TestSomeEmptyStackAndDeepRelease(t *testing.T) {
	t.Parallel()
	var s memory.Store
	base := s.MemoryUsage()
	id := add(t, &s, memory.Some{Element: memory.RootID})
	assertCount(t, &s, memory.RootID, 2)
	for range 20000 {
		next := add(t, &s, memory.Some{Element: id})
		release(t, &s, id)
		id = next
	}
	release(t, &s, id)
	assertCount(t, &s, memory.RootID, 1)
	assertUsage(t, &s, base)
}

func TestInvalidSomeIsAtomic(t *testing.T) {
	t.Parallel()
	var events event.Collector
	s := memory.New(&events)
	base := s.MemoryUsage()
	for _, value := range []memory.Value{memory.Some{Element: 999}, &memory.Some{Element: memory.RootID}, &memory.None{}} {
		if _, err := s.Add(value); err == nil {
			t.Errorf("accepted invalid value %#v", value)
		}
	}
	assertCount(t, s, memory.RootID, 1)
	assertUsage(t, s, base)
	if got := events.Collect(); len(got) != 0 {
		t.Errorf("failed allocations emitted events: %#v", got)
	}
}
