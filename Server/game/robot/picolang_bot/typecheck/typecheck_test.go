package typecheck_test

import (
	"fmt"

	"reflect"
	"slices"
	"testing"

	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/internal/testutil"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/typecheck"
)

type builtinCase struct {
	name   bot.Name
	params []bot.Type
	result bot.Type
}

func builtinCases() []builtinCase {
	b, i, u, terrain, rotation := bot.BoolType{}, bot.IntegerType{}, bot.UnitType{}, bot.TerrainType{}, bot.RotationType{}
	vector := bot.VectorType{ElementType: i}
	return []builtinCase{
		{"not", []bot.Type{b}, b}, {"and", []bot.Type{b, b}, b}, {"or", []bot.Type{b, b}, b},
		{"bool_eq", []bot.Type{b, b}, b}, {"bool_neq", []bot.Type{b, b}, b},
		{"add", []bot.Type{i, i}, i}, {"minus", []bot.Type{i}, i}, {"sub", []bot.Type{i, i}, i},
		{"mul", []bot.Type{i, i}, i}, {"div", []bot.Type{i, i}, i},
		{"integer_eq", []bot.Type{i, i}, b}, {"integer_neq", []bot.Type{i, i}, b},
		{"gt", []bot.Type{i, i}, b}, {"lt", []bot.Type{i, i}, b}, {"geq", []bot.Type{i, i}, b}, {"leq", []bot.Type{i, i}, b},
		{"bool_to_integer", []bot.Type{b}, i}, {"integer_to_bool", []bot.Type{i}, b},
		{"random_bool", nil, b}, {"random_integer", []bot.Type{i, i}, i},
		{"new", nil, vector}, {"push_back", []bot.Type{vector, i}, vector}, {"pop_back", []bot.Type{vector}, vector},
		{"get_back", []bot.Type{vector}, i}, {"size", []bot.Type{vector}, i},
		{"is_land", []bot.Type{terrain}, b}, {"is_goal", []bot.Type{terrain}, b}, {"is_checkpoint", []bot.Type{terrain}, b},
		{"is_own_territory", []bot.Type{terrain}, b}, {"is_enemy_territory", []bot.Type{terrain}, b},
		{"is_luggage", []bot.Type{terrain}, b}, {"is_sea", []bot.Type{terrain}, b}, {"is_rock", []bot.Type{terrain}, b},
		{"forward", nil, b}, {"back", nil, b}, {"rotate", []bot.Type{rotation}, u}, {"wait", []bot.Type{i}, u},
		{"pick_up", nil, b}, {"release", nil, b}, {"drill", nil, b}, {"drill_durability", nil, i},
		{"distance", []bot.Type{bot.DistanceTargetType{}}, i}, {"search_front", nil, terrain}, {"search_now", nil, terrain},
		{"print_integer", []bot.Type{i}, u}, {"print_bool", []bot.Type{b}, u},
	}
}

func builtinProgram(tc builtinCase, argTypes []bot.Type) bot.IndexedProgram {
	params := make([]bot.Param[bot.VariableIndex], len(argTypes))
	args := make([]bot.Expr[bot.VariableIndex, bot.Untyped], len(argTypes))
	for i, argType := range argTypes {
		hint := fmt.Sprintf("arg-%d", i)
		params[i] = bot.Param[bot.VariableIndex]{Source: testutil.Source("param-" + hint), Ident: testutil.Ident("ident-"+hint, bot.VariableIndex(i)), Type: argType}
		args[i] = testutil.Reference(hint, bot.VariableIndex(i))
	}
	program := testutil.MainProgram[bot.VariableIndex](testutil.UnitReturn[bot.VariableIndex]())
	program.Functions["probe"] = bot.Function[bot.VariableIndex, bot.Untyped]{Source: testutil.Source("probe"), Params: params, ReturnType: bot.UnitType{},
		Body: bot.Block[bot.VariableIndex, bot.Untyped]{Source: testutil.Source("probe-body"), Stmts: []bot.Stmt[bot.VariableIndex, bot.Untyped]{
			bot.Declaration[bot.VariableIndex, bot.Untyped]{Source: testutil.Source("result-decl"), Ident: testutil.Ident("result-ident", bot.VariableIndex(len(params))), Type: tc.result, Value: testutil.Call[bot.VariableIndex]("call", tc.name, args...)},
			testutil.UnitReturn[bot.VariableIndex](),
		}},
	}
	return program
}

func TestTypeCheckAllBuiltins(t *testing.T) {
	t.Parallel()
	for _, tc := range builtinCases() {
		t.Run(string(tc.name), func(t *testing.T) {
			t.Parallel()
			program := builtinProgram(tc, tc.params)
			result := testutil.Typed(t, program)
			decl := result.Functions["probe"].Body.Stmts[0].(bot.Declaration[bot.VariableIndex, bot.Type])
			expr := decl.Value.(*bot.CallExpr[bot.VariableIndex, bot.Type])
			testutil.AssertType(t, expr.Annotation, tc.result)
			if expr.HintID != "call" || decl.HintID != "result-decl" || decl.Ident.HintID != "result-ident" {
				t.Error("type checking lost source hints")
			}
			for i, arg := range expr.Args {
				testutil.AssertType(t, arg.(*bot.IdentExpr[bot.VariableIndex, bot.Type]).Annotation, tc.params[i])
			}
			if !reflect.DeepEqual(program, builtinProgram(tc, tc.params)) {
				t.Error("type checking mutated the input")
			}
		})
	}
}

func TestTypeCheckBuiltinArgumentErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range builtinCases() {
		t.Run(string(tc.name), func(t *testing.T) {
			t.Parallel()
			wrongArity := append(slices.Clone(tc.params), bot.UnitType{})
			result, err := typecheck.TypeCheck(builtinProgram(tc, wrongArity))
			if result.Functions != nil {
				t.Error("failed type checking exposed a partial program")
			}
			ds := testutil.Diagnostics(t, err, bot.TypeCheckStage)
			if len(ds) != 1 {
				t.Errorf("argument count error produced secondary diagnostics: %v", ds)
			}
			d := testutil.FindDiagnostic(t, ds, bot.CodeArgumentCount, "probe", "call")
			if d != nil && (d.Expected != fmt.Sprint(len(tc.params)) || d.Actual != fmt.Sprint(len(wrongArity))) {
				t.Errorf("wrong argument counts: %+v", d)
			}
			if len(tc.params) > 0 {
				wrongTypes := slices.Clone(tc.params)
				wrongTypes[0] = bot.UnitType{}
				_, err = typecheck.TypeCheck(builtinProgram(tc, wrongTypes))
				ds := testutil.Diagnostics(t, err, bot.TypeCheckStage)
				if len(ds) != 1 {
					t.Errorf("argument type error produced secondary diagnostics: %v", ds)
				}
				testutil.RequireDiagnostic(t, ds, bot.CodeTypeMismatch, "probe", "arg-0")
			}
		})
	}
}

func TestTypeCheckRemovedVectorBuiltins(t *testing.T) {
	t.Parallel()
	for _, name := range []bot.Name{"get", "set"} {
		t.Run(string(name), func(t *testing.T) {
			t.Parallel()
			args := []bot.Expr[bot.VariableIndex, bot.Untyped]{
				testutil.Call[bot.VariableIndex]("new", "new"),
				testutil.Literal[bot.VariableIndex]("index", bot.IntegerLiteral{}),
			}
			if name == "set" {
				args = append(args, testutil.Literal[bot.VariableIndex]("value", bot.IntegerLiteral{Value: 1}))
			}
			_, err := typecheck.TypeCheck(testutil.MainProgram[bot.VariableIndex](
				testutil.Call[bot.VariableIndex]("removed", name, args...),
				testutil.UnitReturn[bot.VariableIndex](),
			))
			ds := testutil.Diagnostics(t, err, bot.TypeCheckStage)
			if len(ds) != 1 {
				t.Errorf("expected one undefined-function error, got %v", ds)
			}
			testutil.RequireDiagnostic(t, ds, bot.CodeUndefinedFunction, "main", "removed")
		})
	}
}

func TestTypeCheckVectorResultErrors(t *testing.T) {
	t.Parallel()
	i := bot.IntegerType{}
	v := bot.VectorType{ElementType: i}
	for _, tc := range []builtinCase{
		{"push_back", []bot.Type{v, i}, bot.UnitType{}},
		{"pop_back", []bot.Type{v}, bot.UnitType{}},
		{"get_back", []bot.Type{v}, bot.BoolType{}},
	} {
		t.Run(string(tc.name), func(t *testing.T) {
			t.Parallel()
			_, err := typecheck.TypeCheck(builtinProgram(tc, tc.params))
			ds := testutil.Diagnostics(t, err, bot.TypeCheckStage)
			if len(ds) != 1 {
				t.Errorf("expected one result-type error, got %v", ds)
			}
			testutil.RequireDiagnostic(t, ds, bot.CodeTypeMismatch, "probe", "call")
		})
	}
}

func TestTypeCheckExpressionAndControlErrors(t *testing.T) {
	t.Parallel()
	one := testutil.Literal[bot.VariableIndex]("one", bot.IntegerLiteral{Value: 1})
	f := testutil.Literal[bot.VariableIndex]("false", bot.BoolLiteral{})
	program := testutil.MainProgram[bot.VariableIndex](
		bot.Declaration[bot.VariableIndex, bot.Untyped]{Type: bot.IntegerType{}, Ident: testutil.Ident("x", bot.VariableIndex(0)), Value: f},
		bot.Assignment[bot.VariableIndex, bot.Untyped]{Ident: testutil.Ident("assign-x", bot.VariableIndex(0)), Value: testutil.Literal[bot.VariableIndex]("assign-value", bot.BoolLiteral{})},
		bot.If[bot.VariableIndex, bot.Untyped]{Condition: testutil.Literal[bot.VariableIndex]("if-condition", bot.IntegerLiteral{})},
		bot.While[bot.VariableIndex, bot.Untyped]{Condition: testutil.Literal[bot.VariableIndex]("while-condition", bot.IntegerLiteral{})},
		bot.For[bot.VariableIndex, bot.Untyped]{Ident: testutil.Ident("i", bot.VariableIndex(1)), Start: testutil.Literal[bot.VariableIndex]("start", bot.BoolLiteral{}), End: testutil.Literal[bot.VariableIndex]("end", bot.BoolLiteral{}),
			Body: bot.Block[bot.VariableIndex, bot.Untyped]{Stmts: []bot.Stmt[bot.VariableIndex, bot.Untyped]{bot.Assignment[bot.VariableIndex, bot.Untyped]{Ident: testutil.Ident("loop-index", bot.VariableIndex(1)), Value: testutil.Literal[bot.VariableIndex]("loop-value", bot.BoolLiteral{})}}},
		},
		bot.Break[bot.VariableIndex, bot.Untyped]{Source: testutil.Source("break")},
		bot.Continue[bot.VariableIndex, bot.Untyped]{Source: testutil.Source("continue")},
		bot.Return[bot.VariableIndex, bot.Untyped]{Source: testutil.Source("return"), Value: one},
	)
	_, err := typecheck.TypeCheck(program)
	ds := testutil.Diagnostics(t, err, bot.TypeCheckStage)
	if len(ds) != 10 {
		t.Errorf("expected 10 independent errors, got %d:\n%v", len(ds), ds)
	}
	for _, hint := range []bot.HintID{"false", "assign-value", "if-condition", "while-condition", "start", "end", "loop-value", "one"} {
		d := testutil.FindDiagnostic(t, ds, bot.CodeTypeMismatch, "main", hint)
		if d != nil && (d.Expected == "" || d.Actual == "" || d.Expected == d.Actual) {
			t.Errorf("missing expected/actual types: %+v", d)
		}
	}
	testutil.RequireDiagnostic(t, ds, bot.CodeInvalidLoopControl, "main", "break")
	testutil.RequireDiagnostic(t, ds, bot.CodeInvalidLoopControl, "main", "continue")
}

func TestTypeCheckReturnPlacement(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		stmts []bot.Stmt[bot.VariableIndex, bot.Untyped]
	}{
		{"empty", nil},
		{"expression last", []bot.Stmt[bot.VariableIndex, bot.Untyped]{testutil.Literal[bot.VariableIndex]("unit", bot.UnitLiteral{})}},
		{"return before end", []bot.Stmt[bot.VariableIndex, bot.Untyped]{testutil.UnitReturn[bot.VariableIndex](), testutil.Call[bot.VariableIndex]("forward", "forward")}},
		{"both branches return", []bot.Stmt[bot.VariableIndex, bot.Untyped]{bot.If[bot.VariableIndex, bot.Untyped]{Condition: testutil.Literal[bot.VariableIndex]("true", bot.BoolLiteral{Value: true}),
			Then: bot.Block[bot.VariableIndex, bot.Untyped]{Stmts: []bot.Stmt[bot.VariableIndex, bot.Untyped]{testutil.UnitReturn[bot.VariableIndex]()}},
			Else: &bot.Block[bot.VariableIndex, bot.Untyped]{Stmts: []bot.Stmt[bot.VariableIndex, bot.Untyped]{testutil.UnitReturn[bot.VariableIndex]()}},
		}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := typecheck.TypeCheck(testutil.MainProgram(tt.stmts...))
			ds := testutil.Diagnostics(t, err, bot.TypeCheckStage)
			if len(ds) != 1 {
				t.Errorf("expected only missing-return error, got %v", ds)
			}
			testutil.RequireDiagnostic(t, ds, bot.CodeMissingReturn, "main", "body")
		})
	}
}

func TestTypeCheckMainSignature(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		program bot.IndexedProgram
		hint    bot.HintID
	}{
		{"missing", bot.IndexedProgram{}, ""},
		{"return type", bot.IndexedProgram{Functions: map[bot.Name]bot.Function[bot.VariableIndex, bot.Untyped]{"main": {Source: testutil.Source("main"), ReturnType: bot.IntegerType{}, Body: bot.Block[bot.VariableIndex, bot.Untyped]{Stmts: []bot.Stmt[bot.VariableIndex, bot.Untyped]{bot.Return[bot.VariableIndex, bot.Untyped]{Value: testutil.Literal[bot.VariableIndex]("zero", bot.IntegerLiteral{})}}}}}}, "main"},
		{"parameters", bot.IndexedProgram{Functions: map[bot.Name]bot.Function[bot.VariableIndex, bot.Untyped]{"main": {Source: testutil.Source("main"), Params: []bot.Param[bot.VariableIndex]{{Type: bot.BoolType{}, Ident: testutil.Ident("p", bot.VariableIndex(0))}}, ReturnType: bot.UnitType{}, Body: bot.Block[bot.VariableIndex, bot.Untyped]{Stmts: []bot.Stmt[bot.VariableIndex, bot.Untyped]{testutil.UnitReturn[bot.VariableIndex]()}}}}}, "main"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := typecheck.TypeCheck(tt.program)
			testutil.RequireDiagnostic(t, testutil.Diagnostics(t, err, bot.TypeCheckStage), bot.CodeInvalidMain, "main", tt.hint)
		})
	}
}

func TestTypeCheckLiteralsAndPointers(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value bot.Literal
		type_ bot.Type
	}{
		{"bool", &bot.BoolLiteral{Value: true}, &bot.BoolType{}},
		{"integer", &bot.IntegerLiteral{Value: -1}, &bot.IntegerType{}},
		{"unit", &bot.UnitLiteral{}, &bot.UnitType{}},
		{"left", &bot.RotationLiteral{Value: values.Left}, &bot.RotationType{}},
		{"right", bot.RotationLiteral{Value: values.Right}, bot.RotationType{}},
		{"distance target", bot.DistanceTargetLiteral{Value: values.DistanceLuggage}, bot.DistanceTargetType{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value := testutil.Literal[bot.VariableIndex]("literal", tc.value)
			result := testutil.Typed(t, testutil.MainProgram[bot.VariableIndex](&bot.Declaration[bot.VariableIndex, bot.Untyped]{Source: testutil.Source("decl"), Ident: testutil.Ident("x", bot.VariableIndex(0)), Type: tc.type_, Value: &value}, testutil.UnitReturn[bot.VariableIndex]()))
			decl := result.Functions["main"].Body.Stmts[0].(bot.Declaration[bot.VariableIndex, bot.Type])
			testutil.AssertType(t, decl.Value.(*bot.LiteralExpr[bot.VariableIndex, bot.Type]).Annotation, decl.Type)
		})
	}
}
