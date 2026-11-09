package builtin_test

import (
	"errors"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/builtin"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
)

func TestOptionBuiltins(t *testing.T) {
	t.Parallel()
	store, e := setup(t, 0)
	base := store.MemoryUsage()
	none := add(t, store, memory.None{})
	inner := add(t, store, memory.Some{Element: none})
	some := add(t, store, memory.Some{Element: inner})
	empty := add(t, store, memory.Some{Element: memory.RootID})
	for _, tc := range []struct {
		id   memory.ID
		want bool
	}{{none, false}, {inner, true}, {some, true}, {empty, true}} {
		result := immediate(t, e, "is_some", tc.id)
		assertValue(t, store, result, memory.Bool{Value: tc.want})
		release(t, store, result)
	}
	before := store.MemoryUsage()
	if result, err := e.Evaluate("unwrap", []memory.ID{none}); result != nil || !errors.Is(err, builtin.ErrUnwrapNone) {
		t.Errorf("unwrap(none) = %#v, %v", result, err)
	}
	assertUsage(t, store, before)
	unwrapped := immediate(t, e, "unwrap", some)
	if unwrapped != inner {
		t.Errorf("unwrap returned ID %d, want shared ID %d", unwrapped, inner)
	}
	for _, id := range []memory.ID{some, inner, none} {
		release(t, store, id)
	}
	// The returned reference keeps the nested option alive after its owner drops.
	assertValue(t, store, unwrapped, memory.Some{Element: none})
	absent := immediate(t, e, "unwrap", unwrapped)
	release(t, store, unwrapped)
	assertValue(t, store, absent, memory.None{})
	release(t, store, absent)
	root := immediate(t, e, "unwrap", empty)
	assertValue(t, store, root, memory.ListRoot{})
	release(t, store, empty)
	release(t, store, root)
	assertUsage(t, store, base)
}
