package function_test

import (
	"testing"

	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/internal/testutil"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
)

func TestOptionParameterAssignmentAndReturn(t *testing.T) {
	t.Parallel()
	option := bot.OptionType{ElementType: bot.StackType{ElementType: bot.IntegerType{}}}
	p := checked(t, []param{parameter("input", option), parameter("other", option)}, option,
		decl("saved", option, ref("input")), assign("input", ref("other")), ret(ref("saved")))
	env := resources()
	base := env.Memory.MemoryUsage()
	item := add(t, env, memory.Integer{Value: 42})
	list := add(t, env, memory.ListNode{Prev: memory.RootID, Value: item})
	some := add(t, env, memory.Some{Element: list})
	none := add(t, env, memory.None{})
	m := newMachine(t, p, []memory.ID{some, none}, env)
	for _, id := range []memory.ID{some, none, list, item} {
		release(t, env, id)
	}
	result := run(t, m, env, p)
	assertValue(t, env, result, memory.Some{Element: list})
	assertValue(t, env, list, memory.ListNode{Prev: memory.RootID, Value: item})
	release(t, env, result)
	assertUsage(t, env, base)
	var replay testutil.EventReplay
	for _, e := range env.Events.Collect() {
		replay.Apply(t, e)
	}
	replay.AssertEmpty(t)
}

func TestUnwrapOptionParameter(t *testing.T) {
	t.Parallel()
	option := bot.OptionType{ElementType: bot.IntegerType{}}
	p := checked(t, []param{parameter("input", option)}, bot.IntegerType{}, ret(call("unwrap", ref("input"))))
	env := resources()
	base := env.Memory.MemoryUsage()
	item := add(t, env, memory.Integer{Value: 42})
	some := add(t, env, memory.Some{Element: item})
	m := newMachine(t, p, []memory.ID{some}, env)
	release(t, env, some)
	release(t, env, item)
	result := run(t, m, env, p)
	assertValue(t, env, result, memory.Integer{Value: 42})
	release(t, env, result)
	assertUsage(t, env, base)
}
