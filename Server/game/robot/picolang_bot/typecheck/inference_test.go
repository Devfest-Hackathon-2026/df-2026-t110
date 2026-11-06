package typecheck_test

import (
	"testing"

	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/internal/testutil"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/typecheck"
)

func declaration(hint string, slot bot.VariableIndex, typ bot.Type, value bot.Expr[bot.VariableIndex, bot.Untyped]) bot.Declaration[bot.VariableIndex, bot.Untyped] {
	return bot.Declaration[bot.VariableIndex, bot.Untyped]{Source: testutil.Source(hint), Ident: testutil.Ident(hint+"-ident", slot), Type: typ, Value: value}
}

// Inspect the complete result, checking that no internal inference types remain
// and collecting annotations by their original source hints.
func annotations(t *testing.T, program bot.TypedProgram) map[bot.HintID]bot.Type {
	t.Helper()
	result := make(map[bot.HintID]bot.Type)
	var checkType func(bot.Type)
	checkType = func(typ bot.Type) {
		switch typ := typ.(type) {
		case bot.BoolType, bot.IntegerType, bot.UnitType, bot.TerrainType, bot.RotationType:
		case bot.VectorType:
			checkType(typ.ElementType)
		default:
			t.Errorf("non-concrete type escaped checking: %T", typ)
		}
	}
	var visit func(bot.Stmt[bot.VariableIndex, bot.Type])
	block := func(b bot.Block[bot.VariableIndex, bot.Type]) {
		for _, stmt := range b.Stmts {
			visit(stmt)
		}
	}
	record := func(hint bot.HintID, typ bot.Type) {
		checkType(typ)
		result[hint] = typ
	}
	visit = func(stmt bot.Stmt[bot.VariableIndex, bot.Type]) {
		switch n := stmt.(type) {
		case *bot.LiteralExpr[bot.VariableIndex, bot.Type]:
			record(n.HintID, n.Annotation)
		case *bot.IdentExpr[bot.VariableIndex, bot.Type]:
			record(n.HintID, n.Annotation)
		case *bot.CallExpr[bot.VariableIndex, bot.Type]:
			record(n.HintID, n.Annotation)
			for _, arg := range n.Args {
				visit(arg)
			}
		case bot.Declaration[bot.VariableIndex, bot.Type]:
			checkType(n.Type)
			visit(n.Value)
		case bot.Assignment[bot.VariableIndex, bot.Type]:
			visit(n.Value)
		case bot.Return[bot.VariableIndex, bot.Type]:
			visit(n.Value)
		case bot.If[bot.VariableIndex, bot.Type]:
			visit(n.Condition)
			block(n.Then)
			if n.Else != nil {
				block(*n.Else)
			}
		case bot.While[bot.VariableIndex, bot.Type]:
			visit(n.Condition)
			block(n.Body)
		case bot.For[bot.VariableIndex, bot.Type]:
			visit(n.Start)
			visit(n.End)
			block(n.Body)
		case bot.Break[bot.VariableIndex, bot.Type], bot.Continue[bot.VariableIndex, bot.Type]:
		default:
			t.Errorf("unexpected typed node %T", stmt)
		}
	}
	for _, fn := range program.Functions {
		checkType(fn.ReturnType)
		for _, param := range fn.Params {
			checkType(param.Type)
		}
		block(fn.Body)
	}
	return result
}

func TestVectorTypeInference(t *testing.T) {
	t.Parallel()
	vi := bot.VectorType{ElementType: bot.IntegerType{}}
	vb := bot.VectorType{ElementType: bot.BoolType{}}
	vvi, vvb := bot.VectorType{ElementType: vi}, bot.VectorType{ElementType: vb}
	returnContext := testutil.MainProgram[bot.VariableIndex](testutil.UnitReturn[bot.VariableIndex]())
	returnContext.Functions["make_vector"] = bot.Function[bot.VariableIndex, bot.Untyped]{ReturnType: vvb, Body: bot.Block[bot.VariableIndex, bot.Untyped]{Stmts: []bot.Stmt[bot.VariableIndex, bot.Untyped]{
		bot.Return[bot.VariableIndex, bot.Untyped]{Value: testutil.Call[bot.VariableIndex]("return-new", "new")},
	}}}
	argumentContext := testutil.MainProgram[bot.VariableIndex](testutil.Call[bot.VariableIndex]("consume", "consume", testutil.Call[bot.VariableIndex]("argument-new", "new")), testutil.UnitReturn[bot.VariableIndex]())
	argumentContext.Functions["consume"] = bot.Function[bot.VariableIndex, bot.Untyped]{Params: []bot.Param[bot.VariableIndex]{{Ident: testutil.Ident("param", bot.VariableIndex(0)), Type: vi}}, ReturnType: bot.UnitType{}, Body: bot.Block[bot.VariableIndex, bot.Untyped]{Stmts: []bot.Stmt[bot.VariableIndex, bot.Untyped]{testutil.UnitReturn[bot.VariableIndex]()}}}
	tests := []struct {
		name    string
		program bot.IndexedProgram
		want    map[bot.HintID]bot.Type
	}{
		{"declaration", testutil.MainProgram[bot.VariableIndex](declaration("v", 0, vi, testutil.Call[bot.VariableIndex]("new", "new")), testutil.UnitReturn[bot.VariableIndex]()), map[bot.HintID]bot.Type{"new": vi}},
		{"assignment", testutil.MainProgram[bot.VariableIndex](declaration("v", 0, vi, testutil.Call[bot.VariableIndex]("initial", "new")), bot.Assignment[bot.VariableIndex, bot.Untyped]{Ident: testutil.Ident("target", bot.VariableIndex(0)), Value: testutil.Call[bot.VariableIndex]("assigned", "new")}, testutil.UnitReturn[bot.VariableIndex]()), map[bot.HintID]bot.Type{"initial": vi, "assigned": vi}},
		{"return", returnContext, map[bot.HintID]bot.Type{"return-new": vvb}},
		{"user argument", argumentContext, map[bot.HintID]bot.Type{"argument-new": vi}},
		{"later argument", testutil.MainProgram[bot.VariableIndex](testutil.Call[bot.VariableIndex]("push", "push_back", testutil.Call[bot.VariableIndex]("new", "new"), testutil.Literal[bot.VariableIndex]("value", bot.IntegerLiteral{Value: 1})), testutil.UnitReturn[bot.VariableIndex]()), map[bot.HintID]bot.Type{"new": vi, "push": bot.UnitType{}}},
		{"set argument", testutil.MainProgram[bot.VariableIndex](testutil.Call[bot.VariableIndex]("set", "set", testutil.Call[bot.VariableIndex]("new", "new"), testutil.Literal[bot.VariableIndex]("index", bot.IntegerLiteral{}), testutil.Literal[bot.VariableIndex]("value", bot.BoolLiteral{})), testutil.UnitReturn[bot.VariableIndex]()), map[bot.HintID]bot.Type{"new": vb, "set": bot.UnitType{}}},
		{"nested vectors", testutil.MainProgram[bot.VariableIndex](declaration("v", 0, vvb, testutil.Call[bot.VariableIndex]("outer", "new")), testutil.Call[bot.VariableIndex]("push", "push_back", testutil.Reference("v", bot.VariableIndex(0)), testutil.Call[bot.VariableIndex]("inner", "new")), testutil.UnitReturn[bot.VariableIndex]()), map[bot.HintID]bot.Type{"outer": vvb, "inner": vb}},
		{"outer result", testutil.MainProgram[bot.VariableIndex](declaration("v", 0, vi, testutil.Call[bot.VariableIndex]("get", "get", testutil.Call[bot.VariableIndex]("new", "new"), testutil.Literal[bot.VariableIndex]("zero", bot.IntegerLiteral{}))), testutil.UnitReturn[bot.VariableIndex]()), map[bot.HintID]bot.Type{"get": vi, "new": vvi}},
		{"independent instantiations", testutil.MainProgram[bot.VariableIndex](declaration("i", 0, vi, testutil.Call[bot.VariableIndex]("int-new", "new")), declaration("b", 1, vb, testutil.Call[bot.VariableIndex]("bool-new", "new")), testutil.UnitReturn[bot.VariableIndex]()), map[bot.HintID]bot.Type{"int-new": vi, "bool-new": vb}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := annotations(t, testutil.Typed(t, tt.program))
			for hint, expected := range tt.want {
				testutil.AssertType(t, got[hint], expected)
			}
		})
	}
}

func TestVectorInferenceFailures(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		expr bot.Expr[bot.VariableIndex, bot.Untyped]
	}{
		{"standalone", testutil.Call[bot.VariableIndex]("new", "new")},
		{"size", testutil.Call[bot.VariableIndex]("size", "size", testutil.Call[bot.VariableIndex]("new", "new"))},
		{"get", testutil.Call[bot.VariableIndex]("get", "get", testutil.Call[bot.VariableIndex]("new", "new"), testutil.Literal[bot.VariableIndex]("zero", bot.IntegerLiteral{}))},
		{"pop", testutil.Call[bot.VariableIndex]("pop", "pop_back", testutil.Call[bot.VariableIndex]("new", "new"))},
		{"nested unconstrained", testutil.Call[bot.VariableIndex]("push", "push_back", testutil.Call[bot.VariableIndex]("outer-new", "new"), testutil.Call[bot.VariableIndex]("new", "new"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := typecheck.TypeCheck(testutil.MainProgram[bot.VariableIndex](tc.expr, testutil.UnitReturn[bot.VariableIndex]()))
			if result.Functions != nil {
				t.Error("unresolved inference leaked a typed program")
			}
			ds := testutil.Diagnostics(t, err, bot.TypeCheckStage)
			if len(ds) != 1 {
				t.Errorf("expected one primary inference error, got %v", ds)
			}
			testutil.RequireDiagnostic(t, ds, bot.CodeCannotInferType, "main", "new")
		})
	}
}

func TestVectorElementTypeMismatch(t *testing.T) {
	t.Parallel()
	vi := bot.VectorType{ElementType: bot.IntegerType{}}
	program := testutil.MainProgram[bot.VariableIndex](declaration("v", 0, vi, testutil.Call[bot.VariableIndex]("new", "new")),
		testutil.Call[bot.VariableIndex]("push", "push_back", testutil.Reference("v", bot.VariableIndex(0)), testutil.Literal[bot.VariableIndex]("wrong-element", bot.BoolLiteral{})), testutil.UnitReturn[bot.VariableIndex]())
	_, err := typecheck.TypeCheck(program)
	ds := testutil.Diagnostics(t, err, bot.TypeCheckStage)
	if len(ds) != 1 {
		t.Errorf("expected one element-type error, got %v", ds)
	}
	d := testutil.FindDiagnostic(t, ds, bot.CodeTypeMismatch, "main", "wrong-element")
	if d != nil && (d.Expected != "integer" || d.Actual != "bool") {
		t.Errorf("unexpected element-type diagnostic: %+v", d)
	}
}
