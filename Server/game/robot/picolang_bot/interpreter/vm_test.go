package interpreter

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	"github.com/dvpico/Programming-Learning-AR/Server/game/recorder"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/internal/testutil"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/builtin"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/function"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
)

type expr = bot.Expr[bot.Name, bot.Untyped]
type stmt = bot.Stmt[bot.Name, bot.Untyped]
type param = bot.Param[bot.Name]
type fn = bot.Function[bot.Name, bot.Untyped]

// Every source in these fixtures has a unique UUID so the returned event batches
// can also be checked through the production history encoder.
type builder struct{ next uint64 }

func (b *builder) source() bot.Source {
	b.next++
	return bot.Source{HintID: bot.HintID(fmt.Sprintf("00000000-0000-4000-8000-%012x", b.next))}
}

func (b *builder) integer(n int64) expr {
	return bot.LiteralExpr[bot.Name, bot.Untyped]{Source: b.source(), Value: bot.IntegerLiteral{Value: n}}
}

func (b *builder) unit() expr {
	return bot.LiteralExpr[bot.Name, bot.Untyped]{Source: b.source(), Value: bot.UnitLiteral{}}
}

func (b *builder) target(value values.DistanceTarget) expr {
	return bot.LiteralExpr[bot.Name, bot.Untyped]{Source: b.source(), Value: bot.DistanceTargetLiteral{Value: value}}
}

func (b *builder) ref(name bot.Name) expr {
	return bot.IdentExpr[bot.Name, bot.Untyped]{Ident: bot.Ident[bot.Name]{Source: b.source(), Variable: name}}
}

func (b *builder) call(name bot.Name, args ...expr) bot.CallExpr[bot.Name, bot.Untyped] {
	return bot.CallExpr[bot.Name, bot.Untyped]{Source: b.source(), Func: name, Args: args}
}

func (b *builder) ret(value expr) stmt {
	return bot.Return[bot.Name, bot.Untyped]{Source: b.source(), Value: value}
}

func (b *builder) declare(name bot.Name, typ bot.Type, value expr) bot.Declaration[bot.Name, bot.Untyped] {
	return bot.Declaration[bot.Name, bot.Untyped]{Source: b.source(), Ident: bot.Ident[bot.Name]{Source: b.source(), Variable: name}, Type: typ, Value: value}
}

func (b *builder) assign(name bot.Name, value expr) stmt {
	return bot.Assignment[bot.Name, bot.Untyped]{Source: b.source(), Ident: bot.Ident[bot.Name]{Source: b.source(), Variable: name}, Value: value}
}

func (b *builder) block(stmts ...stmt) bot.Block[bot.Name, bot.Untyped] {
	return bot.Block[bot.Name, bot.Untyped]{Source: b.source(), Stmts: stmts}
}

func (b *builder) iff(condition expr, body ...stmt) stmt {
	return bot.If[bot.Name, bot.Untyped]{Source: b.source(), Condition: condition, Then: b.block(body...)}
}

func (b *builder) param(name bot.Name, typ bot.Type) param {
	return param{Source: b.source(), Ident: bot.Ident[bot.Name]{Source: b.source(), Variable: name}, Type: typ}
}

func (b *builder) function(params []param, result bot.Type, body ...stmt) fn {
	return fn{Source: b.source(), Params: params, ReturnType: result, Body: b.block(body...)}
}

func checked(t *testing.T, functions map[bot.Name]fn) bot.TypedProgram {
	t.Helper()
	return testutil.Typed(t, testutil.Indexed(t, bot.ParsedProgram{Functions: functions}))
}

func newVM(t *testing.T, program bot.TypedProgram, seed uint64) *VM {
	t.Helper()
	v, err := New(program, seed)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return v
}

func expectOutput(t *testing.T, output robotmsg.RobotOutput, err error, want robotmsg.Step) {
	t.Helper()
	if err != nil {
		t.Fatalf("execute VM: %v", err)
	}
	if !reflect.DeepEqual(output.Step, want) {
		t.Errorf("step = %#v, want %#v", output.Step, want)
	}
	if _, err := json.Marshal(output.Events); err != nil {
		t.Errorf("events cannot be recorded: %v", err)
	}
}

type vmSnapshot struct {
	Time        uint64
	MemoryBytes uint64
	CallDepth   int
}

func snapshot(v *VM) vmSnapshot {
	return vmSnapshot{Time: v.time.Value(), MemoryBytes: v.memory.MemoryUsage(), CallDepth: len(v.stack)}
}

func assertClean(t *testing.T, v *VM, base uint64) {
	t.Helper()
	usage := snapshot(v)
	if usage.CallDepth != 0 || usage.MemoryBytes != base {
		t.Errorf("VM retained frames or values: %+v, baseline %d", usage, base)
	}
}

func factorial(t *testing.T) (bot.TypedProgram, bot.HintID, bot.HintID, bot.HintID) {
	t.Helper()
	b := &builder{}
	n := b.param("n", bot.IntegerType{})
	recurse := b.call("fact", b.call("sub", b.ref("n"), b.integer(1)))
	fact := b.function([]param{n}, bot.IntegerType{},
		b.iff(b.call("leq", b.ref("n"), b.integer(1)), b.call("print_integer", b.ref("n")), b.ret(b.integer(1))),
		b.ret(b.call("mul", b.ref("n"), recurse)))
	initialCall := b.call("fact", b.integer(6))
	main := b.function(nil, bot.UnitType{}, b.declare("answer", bot.IntegerType{}, initialCall), b.call("print_integer", b.ref("answer")), b.ret(b.unit()))
	return checked(t, map[bot.Name]fn{"main": main, "fact": fact}), n.Ident.HintID, recurse.HintID, initialCall.HintID
}

func TestRecursiveCallsSuspendAndRestoreCallerNodes(t *testing.T) {
	t.Parallel()
	program, parameterHint, recursiveCall, mainCall := factorial(t)
	v := newVM(t, program, 0)
	base := snapshot(v).MemoryBytes
	first, err := v.Start()
	expectOutput(t, first, err, robotmsg.ActionStep{Action: actions.PrintIntegerAction{Integer: 1}})
	if depth := snapshot(v).CallDepth; depth != 7 {
		t.Errorf("depth = %d, want main + 6 invocations", depth)
	}
	before := snapshot(v).Time
	frozen, err := json.Marshal(first.Events)
	if err != nil {
		t.Fatalf("snapshot first batch: %v", err)
	}
	second, err := v.Next(values.UnitValue{})
	expectOutput(t, second, err, robotmsg.ActionStep{Action: actions.PrintIntegerAction{Integer: 720}})
	if usage := snapshot(v); usage.CallDepth != 1 || usage.Time <= before {
		t.Errorf("did not unwind or retain time: %+v", usage)
	}
	restored := 0
	for i, e := range second.Events {
		if drop, ok := e.(robotevent.VariableDropEvent); ok && drop.HintID == parameterHint {
			restored++
			if i+1 >= len(second.Events) {
				t.Error("callee drop has no caller restoration")
				continue
			}
			current, ok := second.Events[i+1].(robotevent.CurrentNodeEvent)
			if !ok || current.HintID != recursiveCall && current.HintID != mainCall {
				t.Errorf("callee returned to %#v", second.Events[i+1])
			}
		}
	}
	if restored != 6 {
		t.Errorf("restored %d callers, want 6", restored)
	}
	last, err := v.Next(values.UnitValue{})
	expectOutput(t, last, err, robotmsg.DoneStep{})
	assertClean(t, v, base)
	if pending := v.Collect(); len(pending) != 0 {
		t.Errorf("events delivered twice: %#v", pending)
	}
	after, err := json.Marshal(first.Events)
	if err != nil || string(after) != string(frozen) {
		t.Error("later execution changed an earlier event batch")
	}
}

func TestVectorsReturnedThroughCallsRemainIndependent(t *testing.T) {
	t.Parallel()
	b := &builder{}
	vector := bot.VectorType{ElementType: bot.IntegerType{}}
	identity := b.function([]param{b.param("input", vector)}, vector, b.ret(b.ref("input")))
	saved := b.declare("saved", vector, b.call("identity", b.ref("v")))
	main := b.function(nil, bot.UnitType{},
		b.declare("v", vector, b.call("new")),
		b.assign("v", b.call("push_back", b.ref("v"), b.integer(1))), saved,
		b.assign("v", b.call("push_back", b.ref("v"), b.integer(2))),
		b.call("print_integer", b.call("size", b.ref("saved"))),
		b.call("print_integer", b.call("get_back", b.ref("v"))), b.ret(b.unit()))
	v := newVM(t, checked(t, map[bot.Name]fn{"main": main, "identity": identity}), 0)
	base := snapshot(v).MemoryBytes
	output, err := v.Start()
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.PrintIntegerAction{Integer: 1}})
	var snapshot values.RecordValue
	for _, e := range output.Events {
		if init, ok := e.(robotevent.VariableInitEvent); ok && init.HintID == saved.Ident.HintID {
			snapshot = init.Value
		}
	}
	output, err = v.Next(values.UnitValue{})
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.PrintIntegerAction{Integer: 2}})
	output, err = v.Next(values.UnitValue{})
	expectOutput(t, output, err, robotmsg.DoneStep{})
	assertClean(t, v, base)
	want := values.VectorRecordValue{Elements: []values.RecordValue{values.IntegerValue{Integer: 1}}}
	if !reflect.DeepEqual(snapshot, want) {
		t.Errorf("saved snapshot = %#v, want %#v", snapshot, want)
	}
}

func TestDeepMutualRecursionUsesExplicitStack(t *testing.T) {
	t.Parallel()
	b := &builder{}
	functions := map[bot.Name]fn{}
	for _, pair := range [][2]bot.Name{{"a", "b"}, {"b", "a"}} {
		functions[pair[0]] = b.function([]param{b.param("n", bot.IntegerType{})}, bot.UnitType{},
			b.iff(b.call("leq", b.ref("n"), b.integer(0)), b.call("print_integer", b.ref("n")), b.ret(b.unit())),
			b.ret(b.call(pair[1], b.call("sub", b.ref("n"), b.integer(1)))))
	}
	functions["main"] = b.function(nil, bot.UnitType{}, b.call("a", b.integer(1000)), b.ret(b.unit()))
	v := newVM(t, checked(t, functions), 0)
	base := v.memory.MemoryUsage()
	output, err := v.Start()
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.PrintIntegerAction{Integer: 0}})
	if depth := len(v.stack); depth != 1002 {
		t.Errorf("call depth = %d, want 1002", depth)
	}
	output, err = v.Next(values.UnitValue{})
	expectOutput(t, output, err, robotmsg.DoneStep{})
	assertClean(t, v, base)
}

func TestRuntimeFailureClosesAllFramesAndDeliversEvents(t *testing.T) {
	t.Parallel()
	b := &builder{}
	vector := bot.VectorType{ElementType: bot.IntegerType{}}
	p := b.param("input", vector)
	local := b.declare("local", bot.IntegerType{}, b.integer(3))
	divide := b.call("div", b.integer(1), b.integer(0))
	bad := b.function([]param{p}, bot.IntegerType{}, local, b.ret(divide))
	caller := b.declare("v", vector, b.call("new"))
	main := b.function(nil, bot.UnitType{}, caller, b.assign("v", b.call("push_back", b.ref("v"), b.integer(1))), b.call("bad", b.ref("v")), b.ret(b.unit()))
	v := newVM(t, checked(t, map[bot.Name]fn{"main": main, "bad": bad}), 0)
	base := snapshot(v).MemoryBytes
	output, err := v.Start()
	var failure *function.RuntimeError
	if !errors.As(err, &failure) || failure.HintID != divide.HintID {
		t.Errorf("runtime failure lost its source: %v", err)
	}
	if output.Step != nil {
		t.Errorf("failed VM returned a successful step: %#v", output.Step)
	}
	var drops []bot.HintID
	for _, e := range output.Events {
		if drop, ok := e.(robotevent.VariableDropEvent); ok {
			drops = append(drops, drop.HintID)
		}
	}
	if want := []bot.HintID{local.Ident.HintID, p.Ident.HintID, caller.Ident.HintID}; !reflect.DeepEqual(drops, want) {
		t.Errorf("cleanup order = %v, want %v", drops, want)
	}
	assertClean(t, v, base)
	if _, err := json.Marshal(output.Events); err != nil {
		t.Errorf("failure events cannot be recorded: %v", err)
	}
	if _, err := v.Next(values.UnitValue{}); !errors.Is(err, ErrState) {
		t.Errorf("failed VM accepted Next: %v", err)
	}
}

func TestLifecycleAndRejectedActionResults(t *testing.T) {
	t.Parallel()
	b := &builder{}
	main := b.function(nil, bot.UnitType{}, b.declare("moved", bot.BoolType{}, b.call("forward")), b.call("print_bool", b.ref("moved")), b.ret(b.unit()))
	v := newVM(t, checked(t, map[bot.Name]fn{"main": main}), 0)
	base := snapshot(v).MemoryBytes
	if _, err := v.Next(values.UnitValue{}); !errors.Is(err, ErrState) {
		t.Errorf("Next before Start: %v", err)
	}
	output, err := v.Start()
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.ForwardAction{}})
	before := snapshot(v)
	if _, err := v.Start(); !errors.Is(err, ErrState) {
		t.Errorf("second Start: %v", err)
	}
	if _, err := v.Next(values.UnitValue{}); err == nil {
		t.Error("Next accepted the wrong action result type")
	}
	if after := snapshot(v); after != before {
		t.Errorf("rejected operation changed usage: before=%+v after=%+v", before, after)
	}
	output, err = v.Next(values.BoolValue{Bool: true})
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.PrintBoolAction{Bool: true}})
	output, err = v.Next(values.UnitValue{})
	expectOutput(t, output, err, robotmsg.DoneStep{})
	assertClean(t, v, base)
	if time := snapshot(v).Time; time != 6 {
		t.Errorf("time = %d, want 6", time)
	}
	if _, err := v.Next(values.UnitValue{}); !errors.Is(err, ErrState) {
		t.Errorf("Next after completion: %v", err)
	}
}

func TestActionResultKindsReachExpressions(t *testing.T) {
	t.Parallel()
	b := &builder{}
	main := b.function(nil, bot.UnitType{},
		b.declare("terrain", bot.TerrainType{}, b.call("search_front")), b.call("print_bool", b.call("is_goal", b.ref("terrain"))),
		b.declare("distance_value", bot.IntegerType{}, b.call("distance", b.target(values.DistanceRock))), b.call("print_integer", b.ref("distance_value")), b.ret(b.unit()))
	v := newVM(t, checked(t, map[bot.Name]fn{"main": main}), 0)
	base := snapshot(v).MemoryBytes
	output, err := v.Start()
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.SearchFrontAction{}})
	output, err = v.Next(values.LandTerrain{Kind: values.GoalLand{}})
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.PrintBoolAction{Bool: true}})
	output, err = v.Next(values.UnitValue{})
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.DistanceAction{Target: values.DistanceRock}})
	output, err = v.Next(values.IntegerValue{Integer: 8})
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.PrintIntegerAction{Integer: 8}})
	output, err = v.Next(values.UnitValue{})
	expectOutput(t, output, err, robotmsg.DoneStep{})
	assertClean(t, v, base)
}

func TestAllDistanceTargetsFlowThroughCallsVectorsAndEvents(t *testing.T) {
	t.Parallel()
	for _, target := range []values.DistanceTarget{
		values.DistanceRobot, values.DistanceAlly, values.DistanceEnemy, values.DistanceRock,
		values.DistanceLuggage, values.DistanceSea, values.DistanceGoal, values.DistanceTerritory,
	} {
		t.Run(string(target), func(t *testing.T) {
			t.Parallel()
			b := &builder{}
			targetType := bot.DistanceTargetType{}
			vector := bot.VectorType{ElementType: targetType}
			nestedType := bot.VectorType{ElementType: vector}
			echo := b.function([]param{b.param("candidate", targetType)}, targetType, b.ret(b.ref("candidate")))
			selected := b.declare("selected", targetType, b.target(values.DistanceRobot))
			nested := b.declare("nested", nestedType, b.call("new"))
			main := b.function(nil, bot.UnitType{}, selected,
				b.assign("selected", b.call("echo_target", b.target(target))),
				b.declare("items", vector, b.call("new")),
				b.assign("items", b.call("push_back", b.ref("items"), b.ref("selected"))), nested,
				b.assign("nested", b.call("push_back", b.ref("nested"), b.ref("items"))),
				b.declare("extracted", targetType, b.call("echo_target", b.call("get_back", b.call("get_back", b.ref("nested"))))),
				b.declare("distance_value", bot.IntegerType{}, b.call("distance", b.ref("extracted"))),
				b.call("print_integer", b.ref("distance_value")), b.ret(b.unit()))
			v := newVM(t, checked(t, map[bot.Name]fn{"main": main, "echo_target": echo}), 0)
			base := snapshot(v).MemoryBytes
			output, err := v.Start()
			expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.DistanceAction{Target: target}})
			initialized, updated, nestedUpdated := false, false, false
			for _, e := range output.Events {
				switch e := e.(type) {
				case robotevent.VariableInitEvent:
					if e.HintID == selected.Ident.HintID {
						initialized = true
						if !reflect.DeepEqual(e.Value, values.DistanceRobot) {
							t.Errorf("distance target initial value = %#v", e.Value)
						}
					}
				case robotevent.VariableUpdateEvent:
					if e.HintID == selected.Ident.HintID {
						updated = true
						data, err := json.Marshal(e)
						want := fmt.Sprintf(`{"kind":"variable_update","hintId":"%s","diff":[{"type":"distance_target","value":%q}]}`, selected.Ident.HintID, target)
						if err != nil || string(data) != want {
							t.Errorf("distance target update = %s, %v; want %s", data, err, want)
						}
					}
					if e.HintID == nested.Ident.HintID {
						nestedUpdated = true
						want := []robotevent.ValueUpdate{robotevent.VectorUpdate{Update: robotevent.VectorReplace{Value: []values.RecordValue{
							values.VectorRecordValue{Elements: []values.RecordValue{target}},
						}}}}
						if !reflect.DeepEqual(e.Diff, want) {
							t.Errorf("nested distance target snapshot = %#v, want %#v", e.Diff, want)
						}
					}
				}
			}
			if !initialized || !updated || !nestedUpdated {
				t.Errorf("missing events: initialized=%t updated=%t nestedUpdated=%t", initialized, updated, nestedUpdated)
			}
			output, err = v.Next(values.IntegerValue{Integer: -1})
			expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.PrintIntegerAction{Integer: -1}})
			output, err = v.Next(values.UnitValue{})
			expectOutput(t, output, err, robotmsg.DoneStep{})
			assertClean(t, v, base)
		})
	}
}

func TestTerrainAttributesSurviveActionResumptionAndReassignment(t *testing.T) {
	t.Parallel()
	b := &builder{}
	vector := bot.VectorType{ElementType: bot.TerrainType{}}
	observed := b.declare("observed", bot.TerrainType{}, b.call("search_front"))
	saved := b.declare("saved", vector, b.call("new"))
	main := b.function(nil, bot.UnitType{}, observed, saved,
		b.assign("saved", b.call("push_back", b.ref("saved"), b.ref("observed"))),
		b.assign("observed", b.call("search_now")),
		b.call("print_bool", b.call("is_luggage", b.call("get_back", b.ref("saved")))),
		b.call("print_bool", b.call("is_own_territory", b.call("get_back", b.ref("saved")))),
		b.call("print_bool", b.call("is_enemy_territory", b.ref("observed"))),
		b.call("print_bool", b.call("is_goal", b.ref("observed"))), b.ret(b.unit()))
	v := newVM(t, checked(t, map[bot.Name]fn{"main": main}), 0)
	base := snapshot(v).MemoryBytes
	output, err := v.Start()
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.SearchFrontAction{}})
	events := append([]robotevent.PicolangEvent(nil), output.Events...)
	own := values.LandTerrain{Kind: values.PlainLand{}, HasLuggage: true, Territory: values.OwnTerritory}
	enemy := values.LandTerrain{Kind: values.PlainLand{}, Territory: values.EnemyTerritory}
	output, err = v.Next(own)
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.SearchNowAction{}})
	events = append(events, output.Events...)
	output, err = v.Next(enemy)
	for _, want := range []bool{true, true, true, false} {
		expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.PrintBoolAction{Bool: want}})
		events = append(events, output.Events...)
		output, err = v.Next(values.UnitValue{})
	}
	expectOutput(t, output, err, robotmsg.DoneStep{})
	assertClean(t, v, base)
	initialized, updated, savedUpdated := false, false, false
	for _, e := range events {
		switch e := e.(type) {
		case robotevent.VariableInitEvent:
			if e.HintID == observed.Ident.HintID {
				initialized = true
				if !reflect.DeepEqual(e.Value, own) {
					t.Errorf("initial terrain snapshot lost attributes: %#v", e.Value)
				}
			}
		case robotevent.VariableUpdateEvent:
			if e.HintID == observed.Ident.HintID {
				updated = true
				if !reflect.DeepEqual(e.Diff, []robotevent.ValueUpdate{robotevent.TerrainUpdate{Value: enemy}}) {
					t.Errorf("terrain replacement lost attributes: %#v", e.Diff)
				}
			}
			if e.HintID == saved.Ident.HintID {
				savedUpdated = true
				want := []robotevent.ValueUpdate{robotevent.VectorUpdate{Update: robotevent.VectorReplace{Value: []values.RecordValue{own}}}}
				if !reflect.DeepEqual(e.Diff, want) {
					t.Errorf("saved terrain snapshot changed after reassignment: %#v", e.Diff)
				}
			}
		}
	}
	if !initialized || !updated || !savedUpdated {
		t.Errorf("missing terrain events: initialized=%t updated=%t savedUpdated=%t", initialized, updated, savedUpdated)
	}
}

func TestSeededStreamSurvivesCallsAndActionSuspensions(t *testing.T) {
	t.Parallel()
	b := &builder{}
	rand := func() expr { return b.call("random_integer", b.integer(math.MinInt64), b.integer(math.MaxInt64)) }
	helper := b.function(nil, bot.IntegerType{}, b.ret(rand()))
	main := b.function(nil, bot.UnitType{}, b.call("print_integer", rand()), b.call("print_integer", b.call("helper")), b.ret(b.unit()))
	program := checked(t, map[bot.Name]fn{"main": main, "helper": helper})
	store := &memory.Store{}
	low, err := store.Add(memory.Integer{Value: math.MinInt64})
	if err != nil {
		t.Fatalf("prepare lower bound: %v", err)
	}
	high, err := store.Add(memory.Integer{Value: math.MaxInt64})
	if err != nil {
		t.Fatalf("prepare upper bound: %v", err)
	}
	for range 2 { // Each VM has independent memory and starts the same seeded stream.
		v := newVM(t, program, 42)
		base := snapshot(v).MemoryBytes
		evaluator, err := builtin.New(store, 42)
		if err != nil {
			t.Fatalf("prepare reference stream: %v", err)
		}
		for i := range 2 {
			var output robotmsg.RobotOutput
			if i == 0 {
				output, err = v.Start()
			} else {
				output, err = v.Next(values.UnitValue{})
			}
			result, evalErr := evaluator.Evaluate("random_integer", []memory.ID{low, high})
			if evalErr != nil {
				t.Fatalf("reference random: %v", evalErr)
			}
			id := result.(builtin.Immediate).Value
			value, _ := store.Get(id)
			expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.PrintIntegerAction{Integer: value.(memory.Integer).Value}})
			if err := store.Release(id); err != nil {
				t.Errorf("release reference result: %v", err)
			}
		}
		output, err := v.Next(values.UnitValue{})
		expectOutput(t, output, err, robotmsg.DoneStep{})
		assertClean(t, v, base)
	}
	for _, id := range []memory.ID{low, high} {
		if err := store.Release(id); err != nil {
			t.Errorf("release bound: %v", err)
		}
	}
}

func TestCloseSuspendedStackReleasesEveryFrame(t *testing.T) {
	t.Parallel()
	program, paramHint, _, _ := factorial(t)
	v := newVM(t, program, 0)
	base := snapshot(v).MemoryBytes
	output, err := v.Start()
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.PrintIntegerAction{Integer: 1}})
	if err := v.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	assertClean(t, v, base)
	drops := 0
	for _, e := range v.Collect() {
		if drop, ok := e.(robotevent.VariableDropEvent); ok && drop.HintID == paramHint {
			drops++
		}
	}
	if drops != 6 {
		t.Errorf("closed %d parameter scopes, want 6", drops)
	}
	if err := v.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
	if len(v.Collect()) != 0 {
		t.Error("Close delivered cleanup events twice")
	}
	if _, err := v.Next(values.UnitValue{}); !errors.Is(err, ErrState) {
		t.Errorf("closed VM accepted Next: %v", err)
	}
}

func TestHubRunsVMAndRecordsItsEvents(t *testing.T) {
	t.Parallel()
	b := &builder{}
	helper := b.function(nil, bot.BoolType{}, b.ret(b.call("forward")))
	main := b.function(nil, bot.UnitType{}, b.declare("moved", bot.BoolType{}, b.call("helper")), b.ret(b.unit()))
	v := newVM(t, checked(t, map[bot.Name]fn{"main": main, "helper": helper}), 0)
	base := snapshot(v).MemoryBytes
	plain := board.NewLandTerrain(board.PlainLand{}, false, 0)
	config, err := board.NewConfig(
		[][]board.Terrain{{plain, plain}},
		map[protocol.RobotID]board.RobotState{"robot": board.NewRobotState(board.NewPosition(0, 0), board.East, 3)},
		10,
		board.AllStoppedRules{},
	)
	if err != nil {
		t.Fatal(err)
	}
	field, err := board.New(config)
	if err != nil {
		t.Fatalf("prepare board: %v", err)
	}
	observedMovement := false
	robot := resumeObserver{VM: v, observe: func() {
		state, exists := field.Snapshot().Robots["robot"]
		if !exists || state.Position() != board.NewPosition(1, 0) {
			t.Errorf("robot did not move before resumption: %+v, exists=%t", state, exists)
		}
		observedMovement = true
	}}
	h, err := hub.New(field, []hub.Participant{{ID: "robot", Robot: robot}})
	if err != nil {
		t.Fatalf("prepare hub: %v", err)
	}
	storage := &recorder.Recorder{}
	report, err := h.Run(hub.Options{MaxResumes: 10, Recorder: storage})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Outcome("robot") != (hub.CompletedOutcome{}) || report.Resumes() != 2 {
		t.Errorf("report = %+v", report)
	}
	if !observedMovement {
		t.Error("movement result did not resume the VM")
	}
	if robots := field.Snapshot().Robots; len(robots) != 0 {
		t.Errorf("finished robots remained on the board: %+v", robots)
	}
	if _, err := json.Marshal(storage.Events); err != nil {
		t.Errorf("history cannot be encoded: %v", err)
	}
	found := false
	for _, e := range storage.Events {
		if e, ok := e.(gameevent.RobotEvent); ok {
			if init, ok := e.Event.(robotevent.VariableInitEvent); ok && init.Value == (values.BoolValue{Bool: true}) {
				found = true
			}
		}
	}
	if !found {
		t.Error("VM event was not delivered to the recorder")
	}
	assertClean(t, v, base)
}

type resumeObserver struct {
	*VM
	observe func()
}

func (r resumeObserver) Next(value values.Value) (robotmsg.RobotOutput, error) {
	r.observe()
	return r.VM.Next(value)
}

func TestEntryPointAndZeroValueValidation(t *testing.T) {
	t.Parallel()
	for _, program := range []bot.TypedProgram{
		{}, {Functions: map[bot.Name]bot.Function[bot.VariableIndex, bot.Type]{"main": {ReturnType: bot.BoolType{}}}},
		{Functions: map[bot.Name]bot.Function[bot.VariableIndex, bot.Type]{"main": {ReturnType: bot.UnitType{}, Params: []bot.Param[bot.VariableIndex]{{Type: bot.IntegerType{}}}}}},
	} {
		if v, err := New(program, 0); err == nil || v != nil {
			t.Errorf("accepted invalid entry point: %#v, %v", v, err)
		}
	}
	var zero VM
	if _, err := zero.Start(); !errors.Is(err, ErrState) {
		t.Errorf("zero Start: %v", err)
	}
	if _, err := zero.Next(values.UnitValue{}); !errors.Is(err, ErrState) {
		t.Errorf("zero Next: %v", err)
	}
	if err := zero.Close(); err != nil {
		t.Errorf("zero Close: %v", err)
	}
}
