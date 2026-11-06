package typecheck_test

import (
	"reflect"
	"testing"

	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/internal/testutil"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/typecheck"
)

func TestTypeCheckMalformedAST(t *testing.T) {
	t.Parallel()
	cycle := &bot.VectorType{}
	cycle.ElementType = cycle
	for _, tc := range []struct {
		name string
		stmt bot.Stmt[bot.VariableIndex, bot.Untyped]
		code bot.DiagnosticCode
		hint bot.HintID
	}{
		{"nil statement", nil, bot.CodeInvalidAST, "body"},
		{"nil pointer statement", (*bot.If[bot.VariableIndex, bot.Untyped])(nil), bot.CodeInvalidAST, "body"},
		{"nil pointer expression", (*bot.CallExpr[bot.VariableIndex, bot.Untyped])(nil), bot.CodeInvalidAST, "body"},
		{"nil return value", bot.Return[bot.VariableIndex, bot.Untyped]{Source: testutil.Source("bad-return")}, bot.CodeInvalidAST, "bad-return"},
		{"nil literal", testutil.Literal[bot.VariableIndex]("literal", nil), bot.CodeInvalidAST, "literal"},
		{"nil pointer literal", testutil.Literal[bot.VariableIndex]("literal", (*bot.BoolLiteral)(nil)), bot.CodeInvalidAST, "literal"},
		{"invalid rotation", testutil.Literal[bot.VariableIndex]("literal", bot.RotationLiteral{Value: "up"}), bot.CodeInvalidAST, "literal"},
		{"invalid distance target", testutil.Literal[bot.VariableIndex]("literal", bot.DistanceTargetLiteral{Value: "other"}), bot.CodeInvalidAST, "literal"},
		{"nil type", declaration("decl", 0, nil, testutil.Literal[bot.VariableIndex]("unit", bot.UnitLiteral{})), bot.CodeInvalidType, "decl"},
		{"nil pointer type", declaration("decl", 0, (*bot.BoolType)(nil), testutil.Literal[bot.VariableIndex]("bool", bot.BoolLiteral{})), bot.CodeInvalidType, "decl"},
		{"missing vector element", declaration("decl", 0, bot.VectorType{}, testutil.Call[bot.VariableIndex]("new", "new")), bot.CodeInvalidType, "decl"},
		{"cyclic vector type", declaration("decl", 0, cycle, testutil.Literal[bot.VariableIndex]("unit", bot.UnitLiteral{})), bot.CodeInvalidType, "decl"},
		{"unknown function", testutil.Call[bot.VariableIndex]("unknown-call", "absent"), bot.CodeUndefinedFunction, "unknown-call"},
		{"undeclared slot", testutil.Reference("invalid-slot", bot.VariableIndex(^uint(0))), bot.CodeInvalidVariableIndex, "invalid-slot"},
		{"assignment slot", bot.Assignment[bot.VariableIndex, bot.Untyped]{Ident: testutil.Ident("target", bot.VariableIndex(0)), Value: testutil.Literal[bot.VariableIndex]("unit", bot.UnitLiteral{})}, bot.CodeInvalidVariableIndex, "target"},
		{"noncontiguous declaration", declaration("decl", 99, bot.UnitType{}, testutil.Literal[bot.VariableIndex]("unit", bot.UnitLiteral{})), bot.CodeInvalidVariableIndex, "decl-ident"},
		{"self initializer", declaration("decl", 0, bot.BoolType{}, testutil.Reference("self", bot.VariableIndex(0))), bot.CodeInvalidVariableIndex, "self"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := typecheck.TypeCheck(testutil.MainProgram[bot.VariableIndex](tc.stmt, testutil.UnitReturn[bot.VariableIndex]()))
			if result.Functions != nil {
				t.Error("malformed AST produced a partial typed program")
			}
			ds := testutil.Diagnostics(t, err, bot.TypeCheckStage)
			testutil.RequireDiagnostic(t, ds, tc.code, "main", tc.hint)
		})
	}
}

func TestTypeCheckSignatureAndParameterErrors(t *testing.T) {
	t.Parallel()
	program := testutil.MainProgram[bot.VariableIndex](testutil.UnitReturn[bot.VariableIndex]())
	program.Functions["probe"] = bot.Function[bot.VariableIndex, bot.Untyped]{Source: testutil.Source("probe"), ReturnType: nil, Params: []bot.Param[bot.VariableIndex]{
		{Source: testutil.Source("bad-type"), Ident: testutil.Ident("bad-slot", bot.VariableIndex(1)), Type: nil},
	}, Body: bot.Block[bot.VariableIndex, bot.Untyped]{Source: testutil.Source("probe-body"), Stmts: []bot.Stmt[bot.VariableIndex, bot.Untyped]{testutil.UnitReturn[bot.VariableIndex]()}}}
	program.Functions["add"] = bot.Function[bot.VariableIndex, bot.Untyped]{Source: testutil.Source("builtin-definition"), ReturnType: bot.UnitType{}, Body: program.Functions["main"].Body}
	program.Functions[""] = bot.Function[bot.VariableIndex, bot.Untyped]{Source: testutil.Source("empty-definition"), ReturnType: bot.UnitType{}, Body: program.Functions["main"].Body}
	_, err := typecheck.TypeCheck(program)
	ds := testutil.Diagnostics(t, err, bot.TypeCheckStage)
	testutil.RequireDiagnostic(t, ds, bot.CodeInvalidName, "", "empty-definition")
	testutil.RequireDiagnostic(t, ds, bot.CodeFunctionConflict, "add", "builtin-definition")
	testutil.RequireDiagnostic(t, ds, bot.CodeInvalidType, "probe", "probe")
	testutil.RequireDiagnostic(t, ds, bot.CodeInvalidType, "probe", "bad-type")
	testutil.RequireDiagnostic(t, ds, bot.CodeInvalidVariableIndex, "probe", "bad-slot")
}

func TestDiagnosticsAreStableAndIncludeAllFunctions(t *testing.T) {
	t.Parallel()
	program := testutil.MainProgram[bot.VariableIndex](testutil.UnitReturn[bot.VariableIndex]())
	for _, name := range []bot.Name{"z", "a"} {
		program.Functions[name] = bot.Function[bot.VariableIndex, bot.Untyped]{Source: testutil.Source(string(name)), ReturnType: nil,
			Body: bot.Block[bot.VariableIndex, bot.Untyped]{Source: testutil.Source(string(name) + "-body"), Stmts: []bot.Stmt[bot.VariableIndex, bot.Untyped]{bot.Break[bot.VariableIndex, bot.Untyped]{Source: testutil.Source(string(name) + "-break")}}},
		}
	}
	_, first := typecheck.TypeCheck(program)
	_, second := typecheck.TypeCheck(program)
	ds := testutil.Diagnostics(t, first, bot.TypeCheckStage)
	if first.Error() != second.Error() {
		t.Errorf("diagnostics are nondeterministic:\n%s\n%s", first, second)
	}
	for _, name := range []bot.Name{"a", "z"} {
		testutil.RequireDiagnostic(t, ds, bot.CodeInvalidType, name, bot.HintID(name))
		testutil.RequireDiagnostic(t, ds, bot.CodeInvalidLoopControl, name, bot.HintID(name)+"-break")
		testutil.RequireDiagnostic(t, ds, bot.CodeMissingReturn, name, bot.HintID(name)+"-body")
	}
	if ds[0].Function != "a" || ds[len(ds)-1].Function != "z" {
		t.Errorf("diagnostics not ordered by function: %v", ds)
	}
}

func TestTypeCheckDoesNotMutateDeclaredPointerTypes(t *testing.T) {
	t.Parallel()
	element := &bot.IntegerType{}
	vector := &bot.VectorType{ElementType: element}
	program := testutil.MainProgram[bot.VariableIndex](declaration("v", 0, vector, testutil.Call[bot.VariableIndex]("new", "new")), testutil.UnitReturn[bot.VariableIndex]())
	result := testutil.Typed(t, program)
	testutil.AssertType(t, annotations(t, result)["new"], bot.VectorType{ElementType: bot.IntegerType{}})
	if vector.ElementType != element || !reflect.DeepEqual(program.Functions["main"].Body.Stmts[0].(bot.Declaration[bot.VariableIndex, bot.Untyped]).Type, vector) {
		t.Error("type checking modified the input type")
	}
}
