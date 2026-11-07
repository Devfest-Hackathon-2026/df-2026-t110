package function_test

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/internal/testutil"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/builtin"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/event"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/function"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
	clock "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/time"
)

type stmt = bot.Stmt[bot.Name, bot.Untyped]
type expr = bot.Expr[bot.Name, bot.Untyped]
type param = bot.Param[bot.Name]

func resources() function.Resources {
	return function.Resources{Memory: &memory.Store{}, Time: &clock.Counter{}, Events: &event.Collector{}}
}

func integer(n int64) expr {
	return testutil.Literal[bot.Name]("integer", bot.IntegerLiteral{Value: n})
}
func boolean(b bool) expr    { return testutil.Literal[bot.Name]("bool", bot.BoolLiteral{Value: b}) }
func ref(name bot.Name) expr { return testutil.Reference(string(name)+"-read", name) }
func call(name bot.Name, args ...expr) expr {
	return testutil.Call[bot.Name](string(name)+"-call", name, args...)
}
func block(stmts ...stmt) bot.Block[bot.Name, bot.Untyped] {
	return bot.Block[bot.Name, bot.Untyped]{Source: testutil.Source("block"), Stmts: stmts}
}
func ret(value expr) stmt {
	return bot.Return[bot.Name, bot.Untyped]{Source: testutil.Source("return"), Value: value}
}
func decl(name bot.Name, typ bot.Type, value expr) stmt {
	return bot.Declaration[bot.Name, bot.Untyped]{Source: testutil.Source("declare"), Ident: testutil.Ident(string(name)+"-definition", name), Type: typ, Value: value}
}
func assign(name bot.Name, value expr) stmt {
	return bot.Assignment[bot.Name, bot.Untyped]{Source: testutil.Source("assign"), Ident: testutil.Ident(string(name)+"-assignment", name), Value: value}
}
func iff(condition expr, then ...stmt) stmt {
	return bot.If[bot.Name, bot.Untyped]{Source: testutil.Source("if"), Condition: condition, Then: block(then...)}
}
func forLoop(name bot.Name, start, end expr, body ...stmt) stmt {
	return bot.For[bot.Name, bot.Untyped]{Source: testutil.Source("for"), Ident: testutil.Ident(string(name)+"-definition", name), Start: start, End: end, Body: block(body...)}
}
func parameter(name bot.Name, typ bot.Type) param {
	return param{Ident: testutil.Ident(string(name)+"-definition", name), Type: typ}
}

func checked(t *testing.T, params []param, result bot.Type, body ...stmt) bot.TypedProgram {
	t.Helper()
	p := testutil.MainProgram[bot.Name](testutil.UnitReturn[bot.Name]())
	p.Functions["probe"] = bot.Function[bot.Name, bot.Untyped]{Source: testutil.Source("probe"), Params: params, ReturnType: result, Body: block(body...)}
	return testutil.Typed(t, testutil.Indexed(t, p))
}

func newMachine(t *testing.T, p bot.TypedProgram, args []memory.ID, env function.Resources) *function.Machine {
	t.Helper()
	m, err := function.New(p.Functions["probe"], args, env)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return m
}

func add(t *testing.T, env function.Resources, value memory.Value) memory.ID {
	t.Helper()
	id, err := env.Memory.Add(value)
	if err != nil {
		t.Fatalf("prepare value: %v", err)
	}
	return id
}

func release(t *testing.T, env function.Resources, id memory.ID) {
	t.Helper()
	if err := env.Memory.Release(id); err != nil {
		t.Errorf("Release(%d): %v", id, err)
	}
}

func boundary(t *testing.T, m *function.Machine) function.State {
	t.Helper()
	for range 10000 {
		state, err := m.Step()
		if err != nil {
			t.Fatalf("Step: %v", err)
		}
		if _, running := state.(function.Running); !running {
			return state
		}
	}
	t.Fatal("machine did not reach a call or return")
	return nil
}

func expectCall(t *testing.T, m *function.Machine, name bot.Name) function.Call {
	t.Helper()
	state := boundary(t, m)
	request, ok := state.(function.Call)
	if !ok || request.Func != name {
		t.Fatalf("state = %#v, want call to %s", state, name)
	}
	return request
}

func resume(t *testing.T, m *function.Machine, env function.Resources, value memory.Value) {
	t.Helper()
	id := add(t, env, value)
	if err := m.Resume(id); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	release(t, env, id)
}

func assertValue(t *testing.T, env function.Resources, id memory.ID, want memory.Value) {
	t.Helper()
	if got, exists := env.Memory.Get(id); !exists || !reflect.DeepEqual(got, want) {
		t.Errorf("Get(%d) = (%#v, %t), want %#v", id, got, exists, want)
	}
}

func assertUsage(t *testing.T, env function.Resources, want uint64) {
	t.Helper()
	if got := env.Memory.MemoryUsage(); got != want {
		t.Errorf("memory usage = %d, want %d", got, want)
	}
}

// This test driver owns dispatch and the call stack. The production machine
// executes no builtins and never recurses into another machine itself.
func run(t *testing.T, root *function.Machine, env function.Resources, p bot.TypedProgram) memory.ID {
	t.Helper()
	evaluator, err := builtin.New(env.Memory, 0)
	if err != nil {
		t.Fatalf("prepare builtin evaluator: %v", err)
	}
	stack := []*function.Machine{root}
	for range 10000 {
		m := stack[len(stack)-1]
		switch state := boundary(t, m).(type) {
		case function.Call:
			if fn, exists := p.Functions[state.Func]; exists {
				callee, err := function.New(fn, state.Args, env)
				if err != nil {
					t.Fatalf("dispatch %s: %v", state.Func, err)
				}
				t.Cleanup(func() {
					if err := callee.Close(); err != nil {
						t.Errorf("close callee: %v", err)
					}
				})
				stack = append(stack, callee)
				continue
			}
			result, err := evaluator.Evaluate(state.Func, state.Args)
			if err != nil {
				t.Fatalf("evaluate builtin %s: %v", state.Func, err)
			}
			immediate, ok := result.(builtin.Immediate)
			if !ok {
				t.Fatalf("unexpected action result %T", result)
			}
			if err := m.Resume(immediate.Value); err != nil {
				t.Fatalf("resume builtin result: %v", err)
			}
			release(t, env, immediate.Value)
		case function.Returned:
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				return state.Value
			}
			if err := stack[len(stack)-1].Resume(state.Value); err != nil {
				t.Fatalf("return to caller: %v", err)
			}
			release(t, env, state.Value)
		default:
			t.Fatalf("unexpected state %T", state)
		}
	}
	t.Fatal("test driver exceeded its call limit")
	return 0
}

func TestParametersAssignmentsAndReturnOwnership(t *testing.T) {
	t.Parallel()
	p := checked(t, []param{parameter("p", bot.IntegerType{})}, bot.IntegerType{},
		decl("saved", bot.IntegerType{}, ref("p")), assign("p", integer(1)), ret(ref("saved")))
	env := resources()
	base := env.Memory.MemoryUsage()
	arg := add(t, env, memory.Integer{Value: 9})
	m := newMachine(t, p, []memory.ID{arg}, env)
	if count, _ := env.Memory.RefCount(arg); count != 2 {
		t.Errorf("parameter was not retained: %d references", count)
	}
	release(t, env, arg)
	result := run(t, m, env, p)
	assertValue(t, env, result, memory.Integer{Value: 9})
	if count, _ := env.Memory.RefCount(result); count != 1 {
		t.Errorf("returned value has %d references, want 1", count)
	}
	if got := env.Time.Value(); got != 6 {
		t.Errorf("time = %d, want 6 (parameter, block, definition, assignment, two reads)", got)
	}
	var variableEvents []robotevent.PicolangEvent
	for _, e := range env.Events.Collect() {
		if _, current := e.(robotevent.CurrentNodeEvent); !current {
			variableEvents = append(variableEvents, e)
		}
	}
	want := []robotevent.PicolangEvent{
		robotevent.VariableInitEvent{HintID: "p-definition", Value: values.IntegerValue{Integer: 9}},
		robotevent.VariableInitEvent{HintID: "saved-definition", Value: values.IntegerValue{Integer: 9}},
		robotevent.VariableUpdateEvent{HintID: "p-definition", Diff: []robotevent.ValueUpdate{robotevent.IntegerUpdate{Value: 1}}},
		robotevent.VariableDropEvent{HintID: "saved-definition"},
		robotevent.VariableDropEvent{HintID: "p-definition"},
	}
	if !reflect.DeepEqual(variableEvents, want) {
		t.Errorf("variable events = %#v, want %#v", variableEvents, want)
	}
	if _, err := m.Step(); err == nil {
		t.Error("finished machine accepted Step")
	}
	if err := m.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	assertValue(t, env, result, memory.Integer{Value: 9})
	release(t, env, result)
	assertUsage(t, env, base)
}

func TestNestedCallsEvaluateArgumentsLeftToRight(t *testing.T) {
	t.Parallel()
	p := checked(t, nil, bot.IntegerType{}, ret(call("add", call("random_integer", integer(1), integer(1)), call("random_integer", integer(2), integer(2)))))
	env := resources()
	base := env.Memory.MemoryUsage()
	m := newMachine(t, p, nil, env)
	first := expectCall(t, m, "random_integer")
	assertValue(t, env, first.Args[0], memory.Integer{Value: 1})
	assertValue(t, env, first.Args[1], memory.Integer{Value: 1})
	alias := first.Args[0]
	first.Args[0] = memory.RootID // The returned slice must not corrupt pending arguments.
	bad := add(t, env, memory.Bool{Value: true})
	before := env.Memory.MemoryUsage()
	if err := m.Resume(bad); err == nil {
		t.Error("Resume accepted a bool for an integer result")
	}
	assertUsage(t, env, before)
	if _, err := m.Step(); err == nil {
		t.Error("suspended machine accepted Step")
	}
	release(t, env, bad)
	if err := env.Memory.Retain(alias); err != nil {
		t.Fatalf("prepare shared result: %v", err)
	}
	if err := m.Resume(alias); err != nil {
		t.Fatalf("Resume shared argument ID: %v", err)
	}
	release(t, env, alias)
	second := expectCall(t, m, "random_integer")
	assertValue(t, env, second.Args[0], memory.Integer{Value: 2})
	resume(t, m, env, memory.Integer{Value: 2})
	outer := expectCall(t, m, "add")
	assertValue(t, env, outer.Args[0], memory.Integer{Value: 1})
	assertValue(t, env, outer.Args[1], memory.Integer{Value: 2})
	resume(t, m, env, memory.Integer{Value: 3})
	result := run(t, m, env, p)
	assertValue(t, env, result, memory.Integer{Value: 3})
	if got := env.Time.Value(); got != 4 {
		t.Errorf("time = %d, want 1 block + 3 calls", got)
	}
	release(t, env, result)
	assertUsage(t, env, base)
}

func TestForLoopsUnwindScopesAndIgnoreIteratorAssignments(t *testing.T) {
	t.Parallel()
	p := checked(t, nil, bot.IntegerType{},
		decl("sum", bot.IntegerType{}, integer(0)),
		forLoop("i", integer(0), integer(4),
			decl("scratch", bot.IntegerType{}, ref("i")),
			iff(call("integer_eq", ref("i"), integer(1)), bot.Continue[bot.Name, bot.Untyped]{}),
			forLoop("j", integer(0), integer(3),
				iff(call("integer_eq", ref("j"), integer(1)), bot.Break[bot.Name, bot.Untyped]{}),
				assign("sum", call("add", ref("sum"), ref("i")))),
			assign("i", integer(99))),
		decl("after", bot.IntegerType{}, integer(7)), ret(call("add", ref("sum"), ref("after"))))
	env := resources()
	base := env.Memory.MemoryUsage()
	result := run(t, newMachine(t, p, nil, env), env, p)
	assertValue(t, env, result, memory.Integer{Value: 12})
	release(t, env, result)
	assertUsage(t, env, base)
	initializations, drops := map[bot.HintID]int{}, map[bot.HintID]int{}
	implicitUpdates := 0
	events := env.Events.Collect()
	for i, e := range events {
		switch e := e.(type) {
		case robotevent.VariableInitEvent:
			initializations[e.HintID]++
		case robotevent.VariableDropEvent:
			drops[e.HintID]++
		case robotevent.VariableUpdateEvent:
			if e.HintID != "i-definition" && e.HintID != "j-definition" {
				continue
			}
			if len(e.Diff) != 1 {
				t.Errorf("unexpected iterator diff: %#v", e.Diff)
				continue
			}
			update, ok := e.Diff[0].(robotevent.IntegerUpdate)
			if !ok || update.Value == 99 { // Ignore the explicit assignment in the body.
				continue
			}
			implicitUpdates++
			if i == 0 {
				t.Error("iterator update has no preceding current-node event")
				continue
			}
			current, ok := events[i-1].(robotevent.CurrentNodeEvent)
			if !ok || current.HintID != "for" {
				t.Errorf("iterator update follows %#v, want current node for", events[i-1])
			}
		}
	}
	if implicitUpdates != 6 {
		t.Errorf("got %d implicit iterator updates, want 6", implicitUpdates)
	}
	if !reflect.DeepEqual(initializations, drops) || initializations["scratch-definition"] != 4 {
		t.Errorf("scope lifetime mismatch: initializations=%v drops=%v", initializations, drops)
	}
}

func TestWhileBreakAndContinue(t *testing.T) {
	t.Parallel()
	p := checked(t, nil, bot.IntegerType{}, decl("n", bot.IntegerType{}, integer(0)), decl("sum", bot.IntegerType{}, integer(0)),
		bot.While[bot.Name, bot.Untyped]{Condition: boolean(true), Body: block(
			decl("old", bot.IntegerType{}, ref("n")),
			assign("n", call("add", ref("n"), integer(1))),
			iff(call("integer_eq", ref("n"), integer(2)), bot.Continue[bot.Name, bot.Untyped]{}),
			iff(call("integer_eq", ref("n"), integer(4)), bot.Break[bot.Name, bot.Untyped]{}),
			assign("sum", call("add", ref("sum"), ref("n")))),
		}, ret(ref("sum")))
	env := resources()
	base := env.Memory.MemoryUsage()
	result := run(t, newMachine(t, p, nil, env), env, p)
	assertValue(t, env, result, memory.Integer{Value: 4})
	release(t, env, result)
	assertUsage(t, env, base)
}

func TestForBounds(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		start, end int64
		count      int64
	}{
		{"empty", 2, 2, 0}, {"reversed", 3, 1, 0}, {"maximum integer", math.MaxInt64 - 1, math.MaxInt64, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := checked(t, nil, bot.IntegerType{}, decl("count", bot.IntegerType{}, integer(0)),
				forLoop("i", integer(tc.start), integer(tc.end), assign("count", call("add", ref("count"), integer(1)))), ret(ref("count")))
			env := resources()
			base := env.Memory.MemoryUsage()
			result := run(t, newMachine(t, p, nil, env), env, p)
			assertValue(t, env, result, memory.Integer{Value: tc.count})
			release(t, env, result)
			assertUsage(t, env, base)
		})
	}
}

func TestRecursiveCallsUseIndependentMachines(t *testing.T) {
	t.Parallel()
	p := checked(t, []param{parameter("n", bot.IntegerType{})}, bot.IntegerType{},
		iff(call("leq", ref("n"), integer(1)), ret(integer(1))),
		ret(call("mul", ref("n"), call("probe", call("sub", ref("n"), integer(1))))))
	env := resources()
	base := env.Memory.MemoryUsage()
	arg := add(t, env, memory.Integer{Value: 6})
	m := newMachine(t, p, []memory.ID{arg}, env)
	release(t, env, arg)
	result := run(t, m, env, p)
	assertValue(t, env, result, memory.Integer{Value: 720})
	release(t, env, result)
	assertUsage(t, env, base)
}

func TestIfElseReturnsFromNestedScope(t *testing.T) {
	t.Parallel()
	for _, condition := range []bool{false, true} {
		t.Run(map[bool]string{true: "then", false: "else"}[condition], func(t *testing.T) {
			t.Parallel()
			other := block(decl("local", bot.IntegerType{}, integer(2)), ret(ref("local")))
			p := checked(t, []param{parameter("condition", bot.BoolType{})}, bot.IntegerType{},
				bot.If[bot.Name, bot.Untyped]{Condition: ref("condition"), Then: block(decl("local", bot.IntegerType{}, integer(1)), ret(ref("local"))), Else: &other}, ret(integer(0)))
			env := resources()
			base := env.Memory.MemoryUsage()
			arg := add(t, env, memory.Bool{Value: condition})
			m := newMachine(t, p, []memory.ID{arg}, env)
			release(t, env, arg)
			result := run(t, m, env, p)
			want := int64(2)
			if condition {
				want = 1
			}
			assertValue(t, env, result, memory.Integer{Value: want})
			release(t, env, result)
			assertUsage(t, env, base)
		})
	}
}

func TestVectorSnapshotsOutliveValuesAndReassignment(t *testing.T) {
	t.Parallel()
	vv := bot.VectorType{ElementType: bot.VectorType{ElementType: bot.IntegerType{}}}
	p := checked(t, []param{parameter("v", vv)}, vv,
		decl("saved", vv, ref("v")), assign("v", call("new")), ret(ref("saved")))
	env := resources()
	base := env.Memory.MemoryUsage()
	one := add(t, env, memory.Integer{Value: 1})
	two := add(t, env, memory.Integer{Value: 2})
	innerFirst := add(t, env, memory.ListNode{Prev: memory.RootID, Value: one})
	inner := add(t, env, memory.ListNode{Prev: innerFirst, Value: two})
	outerFirst := add(t, env, memory.ListNode{Prev: memory.RootID, Value: inner})
	outer := add(t, env, memory.ListNode{Prev: outerFirst, Value: inner})
	release(t, env, one)
	release(t, env, two)
	release(t, env, innerFirst)
	release(t, env, inner)
	release(t, env, outerFirst)
	m := newMachine(t, p, []memory.ID{outer}, env)
	release(t, env, outer)
	result := run(t, m, env, p)
	release(t, env, result)
	assertUsage(t, env, base)
	innerValue := values.VectorRecordValue{Elements: []values.RecordValue{values.IntegerValue{Integer: 1}, values.IntegerValue{Integer: 2}}}
	want := values.VectorRecordValue{Elements: []values.RecordValue{innerValue, innerValue}}
	initCount, updateCount := 0, 0
	for _, e := range env.Events.Collect() {
		switch e := e.(type) {
		case robotevent.VariableInitEvent:
			initCount++
			if !reflect.DeepEqual(e.Value, want) {
				t.Errorf("snapshot changed: %#v", e.Value)
			}
		case robotevent.VariableUpdateEvent:
			updateCount++
			if !reflect.DeepEqual(e.Diff, []robotevent.ValueUpdate{robotevent.VectorUpdate{Update: robotevent.VectorReplace{}}}) {
				t.Errorf("unexpected replacement snapshot: %#v", e.Diff)
			}
		}
	}
	if initCount != 2 || updateCount != 1 {
		t.Errorf("got %d initializations and %d updates", initCount, updateCount)
	}
}

func TestCloseSuspendedAndPartiallyEvaluatedInvocations(t *testing.T) {
	t.Parallel()
	for _, suspend := range []bool{false, true} {
		t.Run(map[bool]string{true: "suspended", false: "partial"}[suspend], func(t *testing.T) {
			t.Parallel()
			p := checked(t, nil, bot.IntegerType{}, ret(call("add", integer(1), call("random_integer", integer(2), integer(2)))))
			env := resources()
			base := env.Memory.MemoryUsage()
			m := newMachine(t, p, nil, env)
			if suspend {
				expectCall(t, m, "random_integer")
			} else {
				for env.Memory.MemoryUsage() == base {
					if _, err := m.Step(); err != nil {
						t.Fatalf("Step: %v", err)
					}
				}
			}
			for range 2 {
				if err := m.Close(); err != nil {
					t.Errorf("Close: %v", err)
				}
			}
			assertUsage(t, env, base)
			if _, err := m.Step(); err == nil {
				t.Error("closed invocation accepted Step")
			}
			if err := m.Resume(memory.RootID); err == nil {
				t.Error("closed invocation accepted Resume")
			}
		})
	}
}

func TestInfiniteLoopRemainsSteppable(t *testing.T) {
	t.Parallel()
	p := checked(t, nil, bot.UnitType{}, bot.While[bot.Name, bot.Untyped]{Condition: boolean(true), Body: block()}, testutil.UnitReturn[bot.Name]())
	env := resources()
	base := env.Memory.MemoryUsage()
	m := newMachine(t, p, nil, env)
	for range 200 {
		if state, err := m.Step(); err != nil || state != (function.Running{}) {
			t.Errorf("loop transition = %#v, %v", state, err)
			break
		}
	}
	if env.Time.Value() < 2 {
		t.Error("loop did not consume block-entry time")
	}
	if err := m.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	assertUsage(t, env, base)
}

func TestRuntimeFailureReleasesReferences(t *testing.T) {
	t.Parallel()
	env := resources()
	base := env.Memory.MemoryUsage()
	fn := bot.Function[bot.VariableIndex, bot.Type]{Body: bot.Block[bot.VariableIndex, bot.Type]{Source: testutil.Source("body"), Stmts: []bot.Stmt[bot.VariableIndex, bot.Type]{
		bot.Declaration[bot.VariableIndex, bot.Type]{Ident: testutil.Ident("local", bot.VariableIndex(0)), Value: bot.LiteralExpr[bot.VariableIndex, bot.Type]{Value: bot.IntegerLiteral{Value: 1}}},
		bot.Break[bot.VariableIndex, bot.Type]{Source: testutil.Source("invalid-break")},
	}}}
	m, err := function.New(fn, nil, env)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for range 30 {
		_, err = m.Step()
		if err != nil {
			break
		}
	}
	var failure *function.RuntimeError
	if !errors.As(err, &failure) || failure.HintID != "invalid-break" || !strings.Contains(err.Error(), "outside a loop") {
		t.Errorf("runtime failure lost source: %v", err)
	}
	assertUsage(t, env, base)
	drops := 0
	for _, e := range env.Events.Collect() {
		if drop, ok := e.(robotevent.VariableDropEvent); ok && drop.HintID == "local" {
			drops++
		}
	}
	if drops != 1 {
		t.Errorf("failed invocation emitted %d drops, want 1", drops)
	}
}

func TestNewAndResumeRejectInvalidInputs(t *testing.T) {
	t.Parallel()
	p := checked(t, []param{parameter("n", bot.IntegerType{})}, bot.IntegerType{}, ret(ref("n")))
	fn := p.Functions["probe"]
	env := resources()
	base := env.Memory.MemoryUsage()
	if _, err := function.New(fn, nil, env); err == nil {
		t.Error("New accepted missing argument")
	}
	if _, err := function.New(fn, []memory.ID{memory.RootID}, env); err == nil {
		t.Error("New accepted wrong argument kind")
	}
	if _, err := function.New(fn, []memory.ID{999}, env); err == nil {
		t.Error("New accepted unknown argument")
	}
	if _, err := function.New(fn, nil, function.Resources{}); err == nil {
		t.Error("New accepted missing resources")
	}
	if events := env.Events.Collect(); len(events) != 0 || env.Time.Value() != 0 {
		t.Error("rejected construction emitted events or charged time")
	}
	assertUsage(t, env, base)
	var zero function.Machine
	if _, err := zero.Step(); err == nil {
		t.Error("zero machine accepted Step")
	}
	if err := zero.Resume(memory.RootID); err == nil {
		t.Error("zero machine accepted Resume")
	}
}

func TestDistanceTargetArgumentsAndCallResults(t *testing.T) {
	t.Parallel()
	vector := bot.VectorType{ElementType: bot.DistanceTargetType{}}
	p := checked(t, []param{parameter("v", vector)}, bot.DistanceTargetType{}, ret(call("get_back", ref("v"))))
	env := resources()
	base := env.Memory.MemoryUsage()
	target := add(t, env, memory.DistanceTarget{Value: values.DistanceEnemy})
	list := add(t, env, memory.ListNode{Prev: memory.RootID, Value: target})
	m := newMachine(t, p, []memory.ID{list}, env)
	release(t, env, target)
	release(t, env, list)
	expectCall(t, m, "get_back")
	wrong := add(t, env, memory.Integer{Value: 1})
	before := env.Memory.MemoryUsage()
	if err := m.Resume(wrong); err == nil {
		t.Error("Resume accepted an integer as a distance target")
	}
	assertUsage(t, env, before)
	release(t, env, wrong)
	resume(t, m, env, memory.DistanceTarget{Value: values.DistanceEnemy})
	result := run(t, m, env, p)
	assertValue(t, env, result, memory.DistanceTarget{Value: values.DistanceEnemy})
	release(t, env, result)
	assertUsage(t, env, base)
}

func TestPrimitiveAssignmentsProduceRecordableEvents(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		typ           bot.Type
		initial, next memory.Value
		update        robotevent.ValueUpdate
	}{
		{"bool", bot.BoolType{}, memory.Bool{}, memory.Bool{Value: true}, robotevent.BoolUpdate{Value: true}},
		{"integer", bot.IntegerType{}, memory.Integer{Value: 1}, memory.Integer{Value: 3}, robotevent.IntegerUpdate{Value: 3}},
		{"unit", bot.UnitType{}, memory.Unit{}, memory.Unit{}, robotevent.UnitUpdate{}},
		{"terrain", bot.TerrainType{}, memory.Terrain{Value: values.LandTerrain{Kind: values.CheckpointLand{Order: 2}}}, memory.Terrain{Value: values.SeaTerrain{}}, robotevent.TerrainUpdate{Value: values.SeaTerrain{}}},
		{"rotation", bot.RotationType{}, memory.Rotation{Value: values.Left}, memory.Rotation{Value: values.Right}, robotevent.RotationUpdate{Value: values.Right}},
		{"distance target", bot.DistanceTargetType{}, memory.DistanceTarget{Value: values.DistanceRobot}, memory.DistanceTarget{Value: values.DistanceTerritory}, robotevent.DistanceTargetUpdate{Value: values.DistanceTerritory}},
		{"terrain attributes", bot.TerrainType{}, memory.Terrain{Value: values.LandTerrain{Kind: values.PlainLand{}, HasLuggage: true, Territory: values.OwnTerritory}}, memory.Terrain{Value: values.LandTerrain{Kind: values.PlainLand{}, Territory: values.EnemyTerritory}}, robotevent.TerrainUpdate{Value: values.LandTerrain{Kind: values.PlainLand{}, Territory: values.EnemyTerritory}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := checked(t, []param{parameter("p", tc.typ), parameter("q", tc.typ)}, tc.typ,
				assign("p", ref("q")), ref("p"), ret(ref("p")))
			fn := p.Functions["probe"]
			fn.Params[0].Ident.HintID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
			fn.Params[1].Ident.HintID = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
			p.Functions["probe"] = fn
			env := resources()
			base := env.Memory.MemoryUsage()
			a, b := add(t, env, tc.initial), add(t, env, tc.next)
			m := newMachine(t, p, []memory.ID{a, b}, env)
			release(t, env, a)
			release(t, env, b)
			result := run(t, m, env, p)
			assertValue(t, env, result, tc.next)
			release(t, env, result)
			assertUsage(t, env, base)
			updates := 0
			for _, e := range env.Events.Collect() {
				if _, current := e.(robotevent.CurrentNodeEvent); current {
					continue
				}
				if _, err := json.Marshal(e); err != nil {
					t.Errorf("event cannot be recorded: %T: %v", e, err)
				}
				if update, ok := e.(robotevent.VariableUpdateEvent); ok {
					updates++
					if update.HintID != fn.Params[0].Ident.HintID || !reflect.DeepEqual(update.Diff, []robotevent.ValueUpdate{tc.update}) {
						t.Errorf("unexpected update: %#v", update)
					}
				}
			}
			if updates != 1 {
				t.Errorf("got %d updates, want 1", updates)
			}
		})
	}
}

func TestBooleanBuiltinsEvaluateActionArgumentsWithoutShortCircuit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name              bot.Name
		left, reply, want bool
	}{
		{"and", false, true, false}, {"or", true, false, true},
	} {
		t.Run(string(tc.name), func(t *testing.T) {
			t.Parallel()
			p := checked(t, nil, bot.BoolType{}, ret(call(tc.name, boolean(tc.left), call("forward"))))
			env := resources()
			base := env.Memory.MemoryUsage()
			m := newMachine(t, p, nil, env)
			evaluator, err := builtin.New(env.Memory, 5)
			if err != nil {
				t.Fatalf("prepare evaluator: %v", err)
			}
			request := expectCall(t, m, "forward")
			result, err := evaluator.Evaluate(request.Func, request.Args)
			if err != nil {
				t.Fatalf("evaluate action: %v", err)
			}
			action, ok := result.(builtin.Action)
			if !ok || action.Action != (actions.ForwardAction{}) {
				t.Fatalf("action = %#v", result)
			}
			// Invalid board output can be rejected without consuming the pending call.
			if _, err := evaluator.Complete(action, values.UnitValue{}); err == nil {
				t.Error("accepted wrong action result type")
			}
			id, err := evaluator.Complete(action, values.BoolValue{Bool: tc.reply})
			if err != nil {
				t.Fatalf("complete action: %v", err)
			}
			if err := m.Resume(id); err != nil {
				t.Fatalf("resume action: %v", err)
			}
			release(t, env, id)
			request = expectCall(t, m, tc.name)
			result, err = evaluator.Evaluate(request.Func, request.Args)
			if err != nil {
				t.Fatalf("evaluate boolean builtin: %v", err)
			}
			value, ok := result.(builtin.Immediate)
			if !ok {
				t.Fatalf("boolean result = %T", result)
			}
			if err := m.Resume(value.Value); err != nil {
				t.Fatalf("resume boolean result: %v", err)
			}
			release(t, env, value.Value)
			returned := run(t, m, env, p)
			assertValue(t, env, returned, memory.Bool{Value: tc.want})
			release(t, env, returned)
			assertUsage(t, env, base)
		})
	}
}

func TestBuiltinVectorUpdatesUseReplaceEvents(t *testing.T) {
	t.Parallel()
	v := bot.VectorType{ElementType: bot.IntegerType{}}
	p := checked(t, nil, bot.IntegerType{},
		decl("v", v, call("new")),
		call("push_back", ref("v"), integer(99)), // A discarded result does not update v.
		assign("v", call("push_back", ref("v"), integer(1))),
		assign("v", call("push_back", ref("v"), integer(2))),
		decl("last", bot.IntegerType{}, call("get_back", ref("v"))),
		assign("v", call("pop_back", ref("v"))),
		ret(call("add", ref("last"), call("size", ref("v")))))
	env := resources()
	base := env.Memory.MemoryUsage()
	result := run(t, newMachine(t, p, nil, env), env, p)
	assertValue(t, env, result, memory.Integer{Value: 3})
	release(t, env, result)
	assertUsage(t, env, base)
	var replacements [][]values.RecordValue
	for _, e := range env.Events.Collect() {
		if update, ok := e.(robotevent.VariableUpdateEvent); ok && update.HintID == "v-definition" {
			if len(update.Diff) != 1 {
				t.Errorf("unexpected vector diff: %#v", update.Diff)
				continue
			}
			vector, ok := update.Diff[0].(robotevent.VectorUpdate)
			if !ok {
				t.Errorf("unexpected update kind: %T", update.Diff[0])
				continue
			}
			replacement, ok := vector.Update.(robotevent.VectorReplace)
			if !ok {
				t.Errorf("vector update is %T, want replace", vector.Update)
				continue
			}
			replacements = append(replacements, replacement.Value)
		}
	}
	want := [][]values.RecordValue{
		{values.IntegerValue{Integer: 1}},
		{values.IntegerValue{Integer: 1}, values.IntegerValue{Integer: 2}},
		{values.IntegerValue{Integer: 1}},
	}
	if !reflect.DeepEqual(replacements, want) {
		t.Errorf("vector snapshots = %#v, want %#v", replacements, want)
	}
}
