package typecheck_test

import (
	"testing"

	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/internal/testutil"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/typecheck"
)

func TestOptionBuiltinTypes(t *testing.T) {
	t.Parallel()
	for _, element := range []bot.Type{bot.BoolType{}, bot.UnitType{}, bot.StackType{ElementType: bot.IntegerType{}}, bot.OptionType{ElementType: bot.BoolType{}}} {
		t.Run(bot.TypeName(element), func(t *testing.T) {
			t.Parallel()
			option := bot.OptionType{ElementType: element}
			for _, tc := range []builtinCase{
				{"some", []bot.Type{element}, option},
				{"none", nil, option},
				{"unwrap", []bot.Type{option}, element},
				{"is_some", []bot.Type{option}, bot.BoolType{}},
			} {
				result := testutil.Typed(t, builtinProgram(tc, tc.params))
				testutil.AssertType(t, annotations(t, result)["call"], tc.result)
			}
		})
	}
}

func TestOptionInferenceFromResult(t *testing.T) {
	t.Parallel()
	option := bot.OptionType{ElementType: bot.IntegerType{}}
	program := testutil.MainProgram[bot.VariableIndex](
		declaration("result", 0, bot.IntegerType{}, testutil.Call[bot.VariableIndex]("unwrap", "unwrap",
			testutil.Call[bot.VariableIndex]("get", "get_back", testutil.Call[bot.VariableIndex]("new", "new")))),
		testutil.UnitReturn[bot.VariableIndex](),
	)
	got := annotations(t, testutil.Typed(t, program))
	testutil.AssertType(t, got["unwrap"], bot.IntegerType{})
	testutil.AssertType(t, got["get"], option)
	testutil.AssertType(t, got["new"], bot.StackType{ElementType: option})
}

func TestOptionInferenceFailures(t *testing.T) {
	t.Parallel()
	for _, name := range []bot.Name{"unwrap", "is_some"} {
		t.Run(string(name), func(t *testing.T) {
			t.Parallel()
			program := testutil.MainProgram[bot.VariableIndex](
				testutil.Call[bot.VariableIndex]("call", name, testutil.Call[bot.VariableIndex]("get", "get_back", testutil.Call[bot.VariableIndex]("new", "new"))),
				testutil.UnitReturn[bot.VariableIndex](),
			)
			_, err := typecheck.TypeCheck(program)
			diagnostics := testutil.Diagnostics(t, err, bot.TypeCheckStage)
			testutil.RequireDiagnostic(t, diagnostics, bot.CodeCannotInferType, "main", "call")
		})
	}
	wrong := builtinCase{"unwrap", []bot.Type{bot.OptionType{ElementType: bot.BoolType{}}}, bot.IntegerType{}}
	if _, err := typecheck.TypeCheck(builtinProgram(wrong, wrong.params)); err == nil {
		t.Error("unwrap(option<bool>) accepted as integer")
	}
	var inference bot.Inference
	sig := inference.Instantiate(bot.BuiltinSignatures()["unwrap"], testutil.Source("unwrap"))
	if bot.UnifyTypes(sig.Result, sig.Params[0]) {
		t.Error("accepted infinite type T = option<T>")
	}
}

func TestOptionCanonicalTypes(t *testing.T) {
	t.Parallel()
	cycle := &bot.OptionType{}
	cycle.ElementType = cycle
	mixed := &bot.OptionType{}
	mixed.ElementType = &bot.StackType{ElementType: mixed}
	for _, typ := range []bot.Type{cycle, mixed, bot.OptionType{}, (*bot.OptionType)(nil)} {
		if _, ok := bot.CanonicalType(typ); ok {
			t.Errorf("accepted malformed type %T", typ)
		}
	}
	inner := &bot.IntegerType{}
	outer := &bot.OptionType{ElementType: &bot.StackType{ElementType: inner}}
	got, ok := bot.CanonicalType(outer)
	if !ok {
		t.Error("rejected valid pointer-based option type")
	}
	testutil.AssertType(t, got, bot.OptionType{ElementType: bot.StackType{ElementType: bot.IntegerType{}}})
	if outer.ElementType.(*bot.StackType).ElementType != inner {
		t.Error("canonicalization mutated the source type")
	}
}

func TestOptionConstructorInference(t *testing.T) {
	t.Parallel()
	i := bot.IntegerType{}
	o := bot.OptionType{ElementType: i}
	nested := bot.OptionType{ElementType: o}
	program := testutil.MainProgram[bot.VariableIndex](
		declaration("absent", 0, o, testutil.Call[bot.VariableIndex]("none", "none")),
		declaration("nested", 1, nested, testutil.Call[bot.VariableIndex]("some", "some", testutil.Call[bot.VariableIndex]("inner-none", "none"))),
		declaration("items", 2, bot.StackType{ElementType: o}, testutil.Call[bot.VariableIndex]("push", "push_back",
			testutil.Call[bot.VariableIndex]("new", "new"), testutil.Call[bot.VariableIndex]("stack-none", "none"))),
		declaration("unwrapped", 3, i, testutil.Call[bot.VariableIndex]("unwrap", "unwrap", testutil.Call[bot.VariableIndex]("context-none", "none"))),
		testutil.UnitReturn[bot.VariableIndex](),
	)
	got := annotations(t, testutil.Typed(t, program))
	for _, hint := range []bot.HintID{"none", "inner-none", "stack-none", "context-none"} {
		testutil.AssertType(t, got[hint], o)
	}
	testutil.AssertType(t, got["some"], nested)
	for _, name := range []bot.Name{"some", "none"} {
		t.Run(string(name)+"_unconstrained", func(t *testing.T) {
			t.Parallel()
			value := testutil.Call[bot.VariableIndex]("none", "none")
			if name == "some" {
				value = testutil.Call[bot.VariableIndex]("some", "some", value)
			}
			_, err := typecheck.TypeCheck(testutil.MainProgram[bot.VariableIndex](value, testutil.UnitReturn[bot.VariableIndex]()))
			ds := testutil.Diagnostics(t, err, bot.TypeCheckStage)
			if len(ds) != 1 || ds[0].Code != bot.CodeCannotInferType {
				t.Errorf("unconstrained constructor diagnostics = %v, want one cannot-infer-type error", ds)
			}
		})
	}
}
