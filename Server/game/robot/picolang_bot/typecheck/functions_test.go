package typecheck_test

import (
	"testing"

	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/internal/testutil"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/typecheck"
)

func TestForwardAndMutualFunctionCalls(t *testing.T) {
	t.Parallel()
	program := testutil.MainProgram[bot.Name](testutil.Call[bot.Name]("main-call", "a", testutil.Literal[bot.Name]("one", bot.IntegerLiteral{Value: 1})), testutil.UnitReturn[bot.Name]())
	for name, target := range map[bot.Name]bot.Name{"a": "z", "z": "a"} {
		program.Functions[name] = bot.Function[bot.Name, bot.Untyped]{Source: testutil.Source(string(name)), ReturnType: bot.IntegerType{}, Params: []bot.Param[bot.Name]{
			{Ident: testutil.Ident(string(name)+"-p", bot.Name("p")), Type: bot.IntegerType{}},
		}, Body: bot.Block[bot.Name, bot.Untyped]{Stmts: []bot.Stmt[bot.Name, bot.Untyped]{
			bot.Return[bot.Name, bot.Untyped]{Value: testutil.Call[bot.Name](string(name)+"-call", target, testutil.Reference(string(name)+"-arg", bot.Name("p")))},
		}}}
	}
	got := annotations(t, testutil.Typed(t, testutil.Indexed(t, program)))
	for _, hint := range []bot.HintID{"main-call", "a-call", "z-call", "a-arg", "z-arg"} {
		testutil.AssertType(t, got[hint], bot.IntegerType{})
	}
}

func TestUserFunctionDiagnostics(t *testing.T) {
	t.Parallel()
	program := testutil.MainProgram[bot.VariableIndex](
		testutil.Call[bot.VariableIndex]("wrong-type", "probe", testutil.Literal[bot.VariableIndex]("arg", bot.BoolLiteral{})),
		testutil.Call[bot.VariableIndex]("wrong-count", "probe"),
		testutil.Call[bot.VariableIndex]("unknown", "missing", testutil.Call[bot.VariableIndex]("unconstrained", "new")),
		testutil.UnitReturn[bot.VariableIndex](),
	)
	program.Functions["probe"] = bot.Function[bot.VariableIndex, bot.Untyped]{ReturnType: bot.IntegerType{}, Params: []bot.Param[bot.VariableIndex]{
		{Ident: testutil.Ident("p", bot.VariableIndex(0)), Type: bot.IntegerType{}},
	}, Body: bot.Block[bot.VariableIndex, bot.Untyped]{Stmts: []bot.Stmt[bot.VariableIndex, bot.Untyped]{
		bot.Return[bot.VariableIndex, bot.Untyped]{Value: testutil.Literal[bot.VariableIndex]("result", bot.BoolLiteral{})},
	}}}
	_, err := typecheck.TypeCheck(program)
	ds := testutil.Diagnostics(t, err, bot.TypeCheckStage)
	if len(ds) != 4 {
		t.Errorf("expected four independent diagnostics without cascades, got %v", ds)
	}
	testutil.RequireDiagnostic(t, ds, bot.CodeArgumentCount, "main", "wrong-count")
	testutil.RequireDiagnostic(t, ds, bot.CodeUndefinedFunction, "main", "unknown")
	for _, location := range []struct {
		fn   bot.Name
		hint bot.HintID
	}{{"main", "arg"}, {"probe", "result"}} {
		d := testutil.FindDiagnostic(t, ds, bot.CodeTypeMismatch, location.fn, location.hint)
		if d != nil && (d.Expected != "integer" || d.Actual != "bool") {
			t.Errorf("wrong user function mismatch: %+v", d)
		}
	}
}
