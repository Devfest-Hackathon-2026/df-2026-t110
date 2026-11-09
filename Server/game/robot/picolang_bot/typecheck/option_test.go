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
