package interpreter

import (
	"testing"

	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
)

func (b *builder) forRange(name bot.Name, start, end expr, body ...stmt) stmt {
	return bot.For[bot.Name, bot.Untyped]{
		Source: b.source(), Ident: bot.Ident[bot.Name]{Source: b.source(), Variable: name},
		Start: start, End: end, Body: b.block(body...),
	}
}

func (b *builder) whileLoop(condition expr, body ...stmt) stmt {
	return bot.While[bot.Name, bot.Untyped]{Source: b.source(), Condition: condition, Body: b.block(body...)}
}

// These examples start with named ASTs and use the production indexer, type
// checker, VM, and builtin evaluator. Action replies simulate the board so the
// expected values and suspension order are explicit, including distance queries
// whose actual board implementation is separate from the VM.
func TestVMProgramScenarios(t *testing.T) {
	t.Parallel()
	type exchange struct {
		action actions.Action
		reply  values.Value
	}
	for _, tc := range []struct {
		name  string
		build func(*builder) map[bot.Name]fn
		trace []exchange
	}{
		{
			name: "arithmetic_and_branch",
			build: func(b *builder) map[bot.Name]fn {
				// result = ((20 + 22) * 2) / 4; print result if 20 <= result < 22.
				otherwise := b.block(b.call("print_integer", b.integer(-1)))
				branch := bot.If[bot.Name, bot.Untyped]{
					Source:    b.source(),
					Condition: b.call("and", b.call("geq", b.ref("result"), b.integer(20)), b.call("lt", b.ref("result"), b.integer(22))),
					Then:      b.block(b.call("print_integer", b.ref("result"))), Else: &otherwise,
				}
				main := b.function(nil, bot.UnitType{},
					b.declare("result", bot.IntegerType{}, b.call("div", b.call("mul", b.call("add", b.integer(20), b.integer(22)), b.integer(2)), b.integer(4))),
					branch, b.call("print_bool", b.call("integer_eq", b.ref("result"), b.integer(21))), b.ret(b.unit()))
				return map[bot.Name]fn{"main": main}
			},
			trace: []exchange{
				{actions.PrintIntegerAction{Integer: 21}, values.UnitValue{}},
				{actions.PrintBoolAction{Bool: true}, values.UnitValue{}},
			},
		},
		{
			name: "recursive_fibonacci",
			build: func(b *builder) map[bot.Name]fn {
				// fib(n) = n for n <= 1, otherwise fib(n-1) + fib(n-2).
				fib := b.function([]param{b.param("n", bot.IntegerType{})}, bot.IntegerType{},
					b.iff(b.call("leq", b.ref("n"), b.integer(1)), b.ret(b.ref("n"))),
					b.ret(b.call("add", b.call("fib", b.call("sub", b.ref("n"), b.integer(1))), b.call("fib", b.call("sub", b.ref("n"), b.integer(2))))))
				main := b.function(nil, bot.UnitType{}, b.call("print_integer", b.call("fib", b.integer(10))), b.ret(b.unit()))
				return map[bot.Name]fn{"main": main, "fib": fib}
			},
			trace: []exchange{{actions.PrintIntegerAction{Integer: 55}, values.UnitValue{}}},
		},
		{
			name: "for_break_continue",
			build: func(b *builder) map[bot.Name]fn {
				// Sum 1..10, skipping 3 and stopping before 8: 1+2+4+5+6+7 = 25.
				main := b.function(nil, bot.UnitType{}, b.declare("total", bot.IntegerType{}, b.integer(0)),
					b.forRange("i", b.integer(1), b.integer(11),
						b.iff(b.call("integer_eq", b.ref("i"), b.integer(3)), bot.Continue[bot.Name, bot.Untyped]{Source: b.source()}),
						b.iff(b.call("integer_eq", b.ref("i"), b.integer(8)), bot.Break[bot.Name, bot.Untyped]{Source: b.source()}),
						b.assign("total", b.call("add", b.ref("total"), b.ref("i")))),
					b.call("print_integer", b.ref("total")), b.ret(b.unit()))
				return map[bot.Name]fn{"main": main}
			},
			trace: []exchange{{actions.PrintIntegerAction{Integer: 25}, values.UnitValue{}}},
		},
		{
			name: "while_across_actions",
			build: func(b *builder) map[bot.Name]fn {
				// Each print suspends the VM; the loop resumes with its counter intact.
				main := b.function(nil, bot.UnitType{}, b.declare("remaining", bot.IntegerType{}, b.integer(3)),
					b.whileLoop(b.call("gt", b.ref("remaining"), b.integer(0)),
						b.call("print_integer", b.ref("remaining")), b.assign("remaining", b.call("sub", b.ref("remaining"), b.integer(1)))),
					b.ret(b.unit()))
				return map[bot.Name]fn{"main": main}
			},
			trace: []exchange{
				{actions.PrintIntegerAction{Integer: 3}, values.UnitValue{}},
				{actions.PrintIntegerAction{Integer: 2}, values.UnitValue{}},
				{actions.PrintIntegerAction{Integer: 1}, values.UnitValue{}},
			},
		},
		{
			name: "vector_argument_value_semantics",
			build: func(b *builder) map[bot.Name]fn {
				// Drain a function-local vector copy to sum [1,2,3,4]; the caller keeps all four elements.
				vector := bot.VectorType{ElementType: bot.IntegerType{}}
				drain := b.function([]param{b.param("items", vector)}, bot.IntegerType{}, b.declare("total", bot.IntegerType{}, b.integer(0)),
					b.whileLoop(b.call("gt", b.call("size", b.ref("items")), b.integer(0)),
						b.assign("total", b.call("add", b.ref("total"), b.call("get_back", b.ref("items")))),
						b.assign("items", b.call("pop_back", b.ref("items")))), b.ret(b.ref("total")))
				main := b.function(nil, bot.UnitType{}, b.declare("items", vector, b.call("new")),
					b.forRange("i", b.integer(1), b.integer(5), b.assign("items", b.call("push_back", b.ref("items"), b.ref("i")))),
					b.call("print_integer", b.call("drain_sum", b.ref("items"))),
					b.call("print_integer", b.call("size", b.ref("items"))), b.ret(b.unit()))
				return map[bot.Name]fn{"main": main, "drain_sum": drain}
			},
			trace: []exchange{
				{actions.PrintIntegerAction{Integer: 10}, values.UnitValue{}},
				{actions.PrintIntegerAction{Integer: 4}, values.UnitValue{}},
			},
		},
		{
			name: "distance_argument_evaluation",
			build: func(b *builder) map[bot.Name]fn {
				// Evaluate enemy before ally, retaining the first result across the second action.
				main := b.function(nil, bot.UnitType{},
					b.call("print_integer", b.call("add", b.call("distance", b.target(values.DistanceEnemy)), b.call("distance", b.target(values.DistanceAlly)))),
					b.ret(b.unit()))
				return map[bot.Name]fn{"main": main}
			},
			trace: []exchange{
				{actions.DistanceAction{Target: values.DistanceEnemy}, values.IntegerValue{Integer: 7}},
				{actions.DistanceAction{Target: values.DistanceAlly}, values.IntegerValue{Integer: 3}},
				{actions.PrintIntegerAction{Integer: 10}, values.UnitValue{}},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := newVM(t, checked(t, tc.build(&builder{})), 0)
			base := v.memory.MemoryUsage()
			output, err := v.Start()
			for _, expected := range tc.trace {
				expectOutput(t, output, err, robotmsg.ActionStep{Action: expected.action})
				output, err = v.Next(expected.reply)
			}
			expectOutput(t, output, err, robotmsg.DoneStep{})
			assertClean(t, v, base)
		})
	}
}
