package function

import (
	"testing"

	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
	clock "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/time"
)

//nolint:paralleltest // AllocsPerRun changes GOMAXPROCS and measures process-wide allocations.
func TestUnloggedVariablesDoNotAllocateSnapshots(t *testing.T) {
	m := &Machine{
		resources: Resources{Memory: &memory.Store{}, Time: &clock.Counter{}},
		owned:     make(map[memory.ID]uint64),
		variables: make([]binding, 0, 1),
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	item := m.allocate(memory.Integer{Value: 42})
	stack := memory.RootID
	for range 64 {
		stack = m.allocate(memory.ListNode{Prev: stack, Value: item})
	}
	nested := m.allocate(memory.ListNode{Prev: memory.RootID, Value: stack})
	ident := bot.Ident[bot.VariableIndex]{Source: bot.Source{HintID: "variable"}, Variable: 0}
	m.define(ident, m.retain(nested))
	for name, run := range map[string]func(){
		"declaration": func() {
			m.drop(m.variables[0].value)
			m.variables = m.variables[:0]
			m.define(ident, m.retain(nested))
		},
		"parameter":  func() { (parameter{index: 0}).run(m) },
		"assignment": func() { m.assign(0, m.retain(nested)) },
	} {
		t.Run(name, func(t *testing.T) {
			if allocations := testing.AllocsPerRun(100, run); allocations != 0 {
				t.Errorf("unlogged variable operation allocated %g times, want zero", allocations)
			}
		})
	}
}
