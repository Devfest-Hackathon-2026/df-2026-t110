package builtin_test

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"testing"

	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/builtin"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
)

func setup(t *testing.T, seed uint64) (*memory.Store, *builtin.Evaluator) {
	t.Helper()
	store := &memory.Store{}
	evaluator, err := builtin.New(store, seed)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return store, evaluator
}

func add(t *testing.T, store *memory.Store, value memory.Value) memory.ID {
	t.Helper()
	id, err := store.Add(value)
	if err != nil {
		t.Fatalf("prepare value: %v", err)
	}
	return id
}

func release(t *testing.T, store *memory.Store, id memory.ID) {
	t.Helper()
	if err := store.Release(id); err != nil {
		t.Errorf("Release(%d): %v", id, err)
	}
}

func immediate(t *testing.T, e *builtin.Evaluator, name bot.Name, args ...memory.ID) memory.ID {
	t.Helper()
	result, err := e.Evaluate(name, args)
	if err != nil {
		t.Fatalf("Evaluate(%s): %v", name, err)
	}
	value, ok := result.(builtin.Immediate)
	if !ok {
		t.Fatalf("Evaluate(%s) = %T, want Immediate", name, result)
	}
	return value.Value
}

func assertValue(t *testing.T, store *memory.Store, id memory.ID, want memory.Value) {
	t.Helper()
	if got, exists := store.Get(id); !exists || !reflect.DeepEqual(got, want) {
		t.Errorf("Get(%d) = (%#v, %t), want %#v", id, got, exists, want)
	}
}

func assertUsage(t *testing.T, store *memory.Store, want uint64) {
	t.Helper()
	if got := store.MemoryUsage(); got != want {
		t.Errorf("memory usage = %d, want %d", got, want)
	}
}

func scalar(t *testing.T, name bot.Name, inputs []memory.Value, want memory.Value) {
	t.Helper()
	store, e := setup(t, 0)
	base := store.MemoryUsage()
	args := make([]memory.ID, len(inputs))
	for i, value := range inputs {
		args[i] = add(t, store, value)
	}
	id := immediate(t, e, name, args...)
	assertValue(t, store, id, want)
	release(t, store, id)
	for _, id := range args {
		release(t, store, id)
	}
	assertUsage(t, store, base)
}

func TestBooleanBuiltins(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ left, right, and, or, equal bool }{
		{false, false, false, false, true}, {false, true, false, true, false},
		{true, false, false, true, false}, {true, true, true, true, true},
	} {
		inputs := []memory.Value{memory.Bool{Value: tc.left}, memory.Bool{Value: tc.right}}
		for name, want := range map[bot.Name]bool{"and": tc.and, "or": tc.or, "bool_eq": tc.equal, "bool_neq": !tc.equal} {
			scalar(t, name, inputs, memory.Bool{Value: want})
		}
		scalar(t, "not", inputs[:1], memory.Bool{Value: !tc.left})
	}
}

func TestIntegerBuiltins(t *testing.T) {
	t.Parallel()
	i := func(n int64) memory.Value { return memory.Integer{Value: n} }
	b := func(value bool) memory.Value { return memory.Bool{Value: value} }
	for _, tc := range []struct {
		name bot.Name
		args []memory.Value
		want memory.Value
	}{
		{"add", []memory.Value{i(7), i(-3)}, i(4)},
		{"minus", []memory.Value{i(-7)}, i(7)},
		{"sub", []memory.Value{i(-7), i(3)}, i(-10)},
		{"mul", []memory.Value{i(-6), i(7)}, i(-42)},
		{"div", []memory.Value{i(-7), i(3)}, i(-2)},
		{"div", []memory.Value{i(7), i(-3)}, i(-2)},
		{"div", []memory.Value{i(math.MinInt64), i(1)}, i(math.MinInt64)},
		{"integer_eq", []memory.Value{i(2), i(2)}, b(true)},
		{"integer_eq", []memory.Value{i(2), i(-2)}, b(false)},
		{"integer_neq", []memory.Value{i(2), i(2)}, b(false)},
		{"integer_neq", []memory.Value{i(2), i(-2)}, b(true)},
		{"gt", []memory.Value{i(math.MaxInt64), i(math.MinInt64)}, b(true)},
		{"gt", []memory.Value{i(2), i(2)}, b(false)},
		{"lt", []memory.Value{i(math.MinInt64), i(math.MaxInt64)}, b(true)},
		{"lt", []memory.Value{i(2), i(2)}, b(false)},
		{"geq", []memory.Value{i(2), i(2)}, b(true)},
		{"geq", []memory.Value{i(1), i(2)}, b(false)},
		{"leq", []memory.Value{i(2), i(2)}, b(true)},
		{"leq", []memory.Value{i(2), i(1)}, b(false)},
	} {
		t.Run(string(tc.name), func(t *testing.T) { t.Parallel(); scalar(t, tc.name, tc.args, tc.want) })
	}
}

func TestTerrainPredicates(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		terrain values.TerrainValue
		want    []bot.Name
	}{
		{"plain", values.LandTerrain{Kind: values.PlainLand{}}, []bot.Name{"is_land"}},
		{"luggage", values.LandTerrain{Kind: values.PlainLand{}, HasLuggage: true}, []bot.Name{"is_land", "is_luggage"}},
		{"own territory", values.LandTerrain{Kind: values.PlainLand{}, Territory: values.OwnTerritory}, []bot.Name{"is_land", "is_own_territory"}},
		{"own territory with luggage", values.LandTerrain{Kind: values.PlainLand{}, Territory: values.OwnTerritory, HasLuggage: true}, []bot.Name{"is_land", "is_own_territory", "is_luggage"}},
		{"enemy territory", values.LandTerrain{Kind: values.PlainLand{}, Territory: values.EnemyTerritory}, []bot.Name{"is_land", "is_enemy_territory"}},
		{"enemy territory with luggage", values.LandTerrain{Kind: values.PlainLand{}, Territory: values.EnemyTerritory, HasLuggage: true}, []bot.Name{"is_land", "is_enemy_territory", "is_luggage"}},
		{"goal", values.LandTerrain{Kind: values.GoalLand{}}, []bot.Name{"is_land", "is_goal"}},
		{"goal with luggage", values.LandTerrain{Kind: values.GoalLand{}, HasLuggage: true}, []bot.Name{"is_land", "is_goal", "is_luggage"}},
		{"checkpoint", values.LandTerrain{Kind: values.CheckpointLand{Order: 3}}, []bot.Name{"is_land", "is_checkpoint"}},
		{"checkpoint with luggage", values.LandTerrain{Kind: values.CheckpointLand{Order: 3}, HasLuggage: true}, []bot.Name{"is_land", "is_checkpoint", "is_luggage"}},
		{"sea", values.SeaTerrain{}, []bot.Name{"is_sea"}}, {"rock", values.RockTerrain{}, []bot.Name{"is_rock"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, name := range []bot.Name{"is_land", "is_goal", "is_own_territory", "is_enemy_territory", "is_checkpoint", "is_luggage", "is_sea", "is_rock"} {
				want := slices.Contains(tc.want, name)
				scalar(t, name, []memory.Value{memory.Terrain{Value: tc.terrain}}, memory.Bool{Value: want})
			}
		})
	}
}

func TestStackValuesShareStorageAndRemainIndependent(t *testing.T) {
	t.Parallel()
	store, e := setup(t, 0)
	base := store.MemoryUsage()
	root := immediate(t, e, "new")
	one, two := add(t, store, memory.Integer{Value: 1}), add(t, store, memory.Integer{Value: 2})
	first := immediate(t, e, "push_back", root, one)
	second := immediate(t, e, "push_back", first, two)
	outer := immediate(t, e, "push_back", root, first)
	inner := immediate(t, e, "get_back", outer)
	modified := immediate(t, e, "push_back", inner, two)
	popped := immediate(t, e, "pop_back", second)
	last := immediate(t, e, "get_back", second)
	if popped != first || last != two || inner != first {
		t.Error("stack copies did not share immutable storage")
	}
	for id, want := range map[memory.ID]int64{root: 0, first: 1, second: 2, outer: 1, modified: 2} {
		size := immediate(t, e, "size", id)
		assertValue(t, store, size, memory.Integer{Value: want})
		release(t, store, size)
	}
	assertValue(t, store, first, memory.ListNode{Prev: root, Value: one})
	assertValue(t, store, second, memory.ListNode{Prev: first, Value: two})
	assertValue(t, store, outer, memory.ListNode{Prev: root, Value: first})
	// Releasing source containers must leave extracted/copied values alive.
	for _, id := range []memory.ID{outer, second, first} {
		release(t, store, id)
	}
	assertValue(t, store, inner, memory.ListNode{Prev: root, Value: one})
	assertValue(t, store, last, memory.Integer{Value: 2})
	for _, id := range []memory.ID{last, popped, modified, inner, two, one, root} {
		release(t, store, id)
	}
	assertUsage(t, store, base)
}

func TestRandomClosedIntervalsAndReproducibility(t *testing.T) {
	t.Parallel()
	a, first := setup(t, 12345)
	b, second := setup(t, 12345)
	for _, bounds := range [][2]int64{{0, 0}, {math.MinInt64, math.MinInt64}, {math.MaxInt64, math.MaxInt64}, {-2, 2}, {math.MinInt64, math.MaxInt64}, {math.MinInt64, 0}, {-1, math.MaxInt64}} {
		argsA := []memory.ID{add(t, a, memory.Integer{Value: bounds[0]}), add(t, a, memory.Integer{Value: bounds[1]})}
		argsB := []memory.ID{add(t, b, memory.Integer{Value: bounds[0]}), add(t, b, memory.Integer{Value: bounds[1]})}
		seen := map[int64]struct{}{}
		for range 500 {
			idA, idB := immediate(t, first, "random_integer", argsA...), immediate(t, second, "random_integer", argsB...)
			valueA, _ := a.Get(idA)
			valueB, _ := b.Get(idB)
			if valueA != valueB {
				t.Error("equal seeds produced different values")
			}
			n := valueA.(memory.Integer).Value
			if n < bounds[0] || n > bounds[1] {
				t.Errorf("random value %d outside %v", n, bounds)
			}
			seen[n] = struct{}{}
			release(t, a, idA)
			release(t, b, idB)
			boolA, boolB := immediate(t, first, "random_bool"), immediate(t, second, "random_bool")
			left, _ := a.Get(boolA)
			right, _ := b.Get(boolB)
			if left != right {
				t.Error("equal seeds produced different bool values")
			}
			release(t, a, boolA)
			release(t, b, boolB)
		}
		if bounds == [2]int64{-2, 2} && len(seen) != 5 {
			t.Errorf("closed interval did not produce all five values: %v", seen)
		}
		for _, id := range argsA {
			release(t, a, id)
		}
		for _, id := range argsB {
			release(t, b, id)
		}
	}
}

func TestActionRequestsAndResults(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   bot.Name
		args   []memory.Value
		action actions.Action
		reply  values.Value
		want   memory.Value
	}{
		{"forward", nil, actions.ForwardAction{}, values.BoolValue{Bool: true}, memory.Bool{Value: true}},
		{"back", nil, actions.BackAction{}, values.BoolValue{}, memory.Bool{}},
		{"rotate", []memory.Value{memory.Rotation{Value: values.Left}}, actions.RotateAction{Rotation: values.Left}, values.UnitValue{}, memory.Unit{}},
		{"rotate", []memory.Value{memory.Rotation{Value: values.Right}}, actions.RotateAction{Rotation: values.Right}, values.UnitValue{}, memory.Unit{}},
		{"wait", []memory.Value{memory.Integer{Value: 7}}, actions.WaitAction{Integer: 7}, values.UnitValue{}, memory.Unit{}},
		{"pick_up", nil, actions.PickUpAction{}, values.BoolValue{Bool: true}, memory.Bool{Value: true}},
		{"release", nil, actions.ReleaseAction{}, values.BoolValue{}, memory.Bool{}},
		{"drill", nil, actions.DrillAction{}, values.BoolValue{Bool: true}, memory.Bool{Value: true}},
		{"drill_durability", nil, actions.DrillDurabilityAction{}, values.IntegerValue{Integer: 3}, memory.Integer{Value: 3}},
		{"distance", []memory.Value{memory.DistanceTarget{Value: values.DistanceEnemy}}, actions.DistanceAction{Target: values.DistanceEnemy}, values.IntegerValue{Integer: 42}, memory.Integer{Value: 42}},
		{"search_front", nil, actions.SearchFrontAction{}, values.LandTerrain{Kind: values.CheckpointLand{Order: 2}}, memory.Terrain{Value: values.LandTerrain{Kind: values.CheckpointLand{Order: 2}}}},
		{"search_now", nil, actions.SearchNowAction{}, values.SeaTerrain{}, memory.Terrain{Value: values.SeaTerrain{}}},
		{"search_now", nil, actions.SearchNowAction{}, values.LandTerrain{Kind: values.PlainLand{}, HasLuggage: true, Territory: values.OwnTerritory}, memory.Terrain{Value: values.LandTerrain{Kind: values.PlainLand{}, HasLuggage: true, Territory: values.OwnTerritory}}},
		{"print_integer", []memory.Value{memory.Integer{Value: -42}}, actions.PrintIntegerAction{Integer: -42}, values.UnitValue{}, memory.Unit{}},
		{"print_bool", []memory.Value{memory.Bool{Value: true}}, actions.PrintBoolAction{Bool: true}, values.UnitValue{}, memory.Unit{}},
	} {
		t.Run(string(tc.name), func(t *testing.T) {
			t.Parallel()
			store, e := setup(t, 0)
			base := store.MemoryUsage()
			args := make([]memory.ID, len(tc.args))
			for i, value := range tc.args {
				args[i] = add(t, store, value)
			}
			before := store.MemoryUsage()
			result, err := e.Evaluate(tc.name, args)
			if err != nil {
				t.Fatalf("Evaluate: %v", err)
			}
			request, ok := result.(builtin.Action)
			if !ok {
				t.Fatalf("result = %T, want Action", result)
			}
			if !reflect.DeepEqual(request.Action, tc.action) {
				t.Errorf("action = %#v, want %#v", request.Action, tc.action)
			}
			assertUsage(t, store, before)
			id, err := e.Complete(request, tc.reply)
			if err != nil {
				t.Fatalf("Complete: %v", err)
			}
			assertValue(t, store, id, tc.want)
			release(t, store, id)
			for i, id := range args {
				assertValue(t, store, id, tc.args[i])
				release(t, store, id)
			}
			assertUsage(t, store, base)
		})
	}
}

func TestEveryBuiltinSignatureHasAHandler(t *testing.T) {
	t.Parallel()
	for name, signature := range bot.BuiltinSignatures() {
		t.Run(string(name), func(t *testing.T) {
			t.Parallel()
			store, e := setup(t, 0)
			base := store.MemoryUsage()
			one := add(t, store, memory.Integer{Value: 1})
			stack := add(t, store, memory.ListNode{Prev: memory.RootID, Value: one})
			owned := []memory.ID{one, stack}
			args := make([]memory.ID, len(signature.Params))
			for i, typ := range signature.Params {
				var value memory.Value
				switch typ.(type) {
				case bot.BoolType:
					value = memory.Bool{Value: true}
				case bot.IntegerType:
					value = memory.Integer{Value: 1}
				case bot.RotationType:
					value = memory.Rotation{Value: values.Left}
				case bot.DistanceTargetType:
					value = memory.DistanceTarget{Value: values.DistanceTerritory}
				case bot.TerrainType:
					value = memory.Terrain{Value: values.SeaTerrain{}}
				case bot.StackType:
					args[i] = stack
					continue
				default:
					args[i] = one
					continue
				}
				args[i] = add(t, store, value)
				owned = append(owned, args[i])
			}
			result, err := e.Evaluate(name, args)
			if err != nil {
				t.Fatalf("signature has no working handler: %v", err)
			}
			switch result := result.(type) {
			case builtin.Immediate:
				release(t, store, result.Value)
			case builtin.Action:
				if !actions.ValidAction(result.Action) {
					t.Errorf("invalid action: %T", result.Action)
				}
			default:
				t.Errorf("invalid result %T", result)
			}
			for _, id := range owned {
				release(t, store, id)
			}
			assertUsage(t, store, base)
		})
	}
}

func TestErrorsDoNotMutateArgumentsOrAllocate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name bot.Name
		args []memory.Value
	}{
		{"div", []memory.Value{memory.Integer{Value: 1}, memory.Integer{}}},
		{"random_integer", []memory.Value{memory.Integer{Value: 2}, memory.Integer{Value: 1}}},
		{"pop_back", []memory.Value{memory.ListRoot{}}}, {"get_back", []memory.Value{memory.ListRoot{}}},
		{"and", []memory.Value{memory.Bool{}, memory.Integer{Value: 1}}},
		{"or", []memory.Value{memory.Bool{Value: true}, memory.Integer{Value: 1}}},
		{"add", []memory.Value{memory.Integer{Value: 1}}}, {"size", []memory.Value{memory.Unit{}}},
		{"rotate", []memory.Value{memory.Integer{Value: 1}}},
		{"distance", nil},
		{"distance", []memory.Value{memory.Integer{Value: 1}}},
		{"distance", []memory.Value{memory.Rotation{Value: values.Left}}},
	} {
		t.Run(string(tc.name), func(t *testing.T) {
			t.Parallel()
			store, e := setup(t, 0)
			args := make([]memory.ID, len(tc.args))
			for i, value := range tc.args {
				args[i] = add(t, store, value)
			}
			before := store.MemoryUsage()
			if result, err := e.Evaluate(tc.name, args); err == nil || result != nil {
				t.Errorf("invalid call returned %#v, %v", result, err)
			}
			assertUsage(t, store, before)
			for i, id := range args {
				assertValue(t, store, id, tc.args[i])
				release(t, store, id)
			}
		})
	}
	_, e := setup(t, 0)
	if _, err := e.Evaluate("user_function", nil); !errors.Is(err, builtin.ErrUnknownFunction) {
		t.Errorf("unknown function: %v", err)
	}
	if _, err := e.Evaluate("is_laggage", nil); !errors.Is(err, builtin.ErrUnknownFunction) {
		t.Errorf("obsolete terrain predicate is still recognized: %v", err)
	}
	for _, name := range []bot.Name{"bool_to_integer", "integer_to_bool"} {
		if _, err := e.Evaluate(name, nil); !errors.Is(err, builtin.ErrUnknownFunction) {
			t.Errorf("removed cast %q is still recognized: %v", name, err)
		}
	}
	if _, err := e.Evaluate("not", []memory.ID{999}); err == nil {
		t.Error("unknown argument ID accepted")
	}
	if _, err := builtin.New(nil, 0); err == nil {
		t.Error("nil store accepted")
	}
	var zero builtin.Evaluator
	if _, err := zero.Evaluate("new", nil); err == nil {
		t.Error("uninitialized evaluator accepted")
	}
}

func TestCompleteRejectsMalformedAndWrongKindResults(t *testing.T) {
	t.Parallel()
	store, e := setup(t, 0)
	result, err := e.Evaluate("forward", nil)
	if err != nil {
		t.Fatalf("prepare action: %v", err)
	}
	request := result.(builtin.Action)
	before := store.MemoryUsage()
	for _, value := range []values.Value{nil, (*values.BoolValue)(nil), &values.BoolValue{}, values.IntegerValue{Integer: 1}, values.UnitValue{}, values.LandTerrain{}, values.LandTerrain{Kind: &values.PlainLand{}}} {
		if _, err := e.Complete(request, value); err == nil {
			t.Errorf("accepted invalid result %#v", value)
		}
		assertUsage(t, store, before)
	}
	if _, err := e.Complete(builtin.Action{}, values.BoolValue{}); err == nil {
		t.Error("accepted missing action")
	}
}
