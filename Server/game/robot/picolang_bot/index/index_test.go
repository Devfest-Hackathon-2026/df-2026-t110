package index_test

import (
	"reflect"
	"testing"

	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/index"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/internal/testutil"
)

func scopeFixture[V bot.Variable](variables map[string]V) bot.Program[V, bot.Untyped] {
	x, y, loop, after, param := variables["x"], variables["y"], variables["loop"], variables["after"], variables["param"]
	declare := func(hint string, variable V) bot.Declaration[V, bot.Untyped] {
		return bot.Declaration[V, bot.Untyped]{Source: testutil.Source(hint), Type: bot.IntegerType{}, Ident: testutil.Ident(hint+"-ident", variable), Value: testutil.Literal[V](hint+"-value", bot.IntegerLiteral{Value: 1})}
	}
	program := testutil.MainProgram[V](
		declare("x-decl", x),
		bot.If[V, bot.Untyped]{Source: testutil.Source("if"), Condition: testutil.Call[V]("forward", "forward"),
			Then: bot.Block[V, bot.Untyped]{Source: testutil.Source("then"), Stmts: []bot.Stmt[V, bot.Untyped]{declare("then-y", y)}},
			Else: &bot.Block[V, bot.Untyped]{Source: testutil.Source("else"), Stmts: []bot.Stmt[V, bot.Untyped]{declare("else-y", y)}},
		},
		declare("outer-y", y),
		bot.Assignment[V, bot.Untyped]{Source: testutil.Source("assign"), Ident: testutil.Ident("assign-x", x), Value: testutil.Reference("read-y", y)},
		bot.For[V, bot.Untyped]{Source: testutil.Source("for"), Ident: testutil.Ident("loop-ident", loop), Start: testutil.Reference("start", x), End: testutil.Reference("end", y),
			Body: bot.Block[V, bot.Untyped]{Source: testutil.Source("for-body"), Stmts: []bot.Stmt[V, bot.Untyped]{
				declare("inner-after", after), testutil.Reference("read-loop", loop), bot.Continue[V, bot.Untyped]{Source: testutil.Source("continue")},
			}},
		},
		declare("after-loop", loop),
		bot.While[V, bot.Untyped]{Source: testutil.Source("while"), Condition: testutil.Literal[V]("false", bot.BoolLiteral{}),
			Body: bot.Block[V, bot.Untyped]{Source: testutil.Source("while-body"), Stmts: []bot.Stmt[V, bot.Untyped]{bot.Break[V, bot.Untyped]{Source: testutil.Source("break")}}},
		},
		testutil.Call[V]("helper-call", "helper", testutil.Literal[V]("true", bot.BoolLiteral{Value: true})),
		testutil.UnitReturn[V](),
	)
	program.Functions["helper"] = bot.Function[V, bot.Untyped]{Source: testutil.Source("helper"), Params: []bot.Param[V]{{Source: testutil.Source("param"), Ident: testutil.Ident("param-ident", param), Type: bot.BoolType{}}}, ReturnType: bot.BoolType{},
		Body: bot.Block[V, bot.Untyped]{Source: testutil.Source("helper-body"), Stmts: []bot.Stmt[V, bot.Untyped]{
			bot.Return[V, bot.Untyped]{Source: testutil.Source("helper-return"), Value: testutil.Call[V]("recurse", "helper", testutil.Reference("read-param", param))},
		}},
	}
	return program
}

func TestIndexScopesAndPreservesAST(t *testing.T) {
	t.Parallel()
	names := map[string]bot.Name{"x": "x", "y": "y", "loop": "i", "after": "after", "param": "p"}
	input := scopeFixture(names)
	want := scopeFixture(map[string]bot.VariableIndex{"x": 0, "y": 1, "loop": 2, "after": 3, "param": 0})
	result := testutil.Indexed(t, input)
	if !reflect.DeepEqual(result, want) {
		t.Errorf("indexed AST differs from expected scopes, slots, or source hints:\ngot %#v\nwant %#v", result, want)
	}
	if !reflect.DeepEqual(input, scopeFixture(names)) {
		t.Error("indexing modified the source AST")
	}
	testutil.Typed(t, result)
}

func TestIndexNameErrors(t *testing.T) {
	t.Parallel()
	declare := func(hint string, name bot.Name, value bot.Expr[bot.Name, bot.Untyped]) bot.Declaration[bot.Name, bot.Untyped] {
		return bot.Declaration[bot.Name, bot.Untyped]{Source: testutil.Source(hint), Type: bot.IntegerType{}, Ident: testutil.Ident(hint+"-ident", name), Value: value}
	}
	one := testutil.Literal[bot.Name]("one", bot.IntegerLiteral{Value: 1})
	tests := []struct {
		name    string
		program bot.ParsedProgram
		code    bot.DiagnosticCode
		hint    bot.HintID
	}{
		{"undefined reference", testutil.MainProgram[bot.Name](testutil.Reference("missing", bot.Name("x"))), bot.CodeUndefinedVariable, "missing"},
		{"undefined assignment", testutil.MainProgram[bot.Name](bot.Assignment[bot.Name, bot.Untyped]{Ident: testutil.Ident("target", bot.Name("x")), Value: one}), bot.CodeUndefinedVariable, "target"},
		{"self initializer", testutil.MainProgram[bot.Name](declare("x", "x", testutil.Reference("self", bot.Name("x")))), bot.CodeUndefinedVariable, "self"},
		{"read before declaration", testutil.MainProgram[bot.Name](testutil.Reference("early", bot.Name("x")), declare("x", "x", one)), bot.CodeUndefinedVariable, "early"},
		{"duplicate", testutil.MainProgram[bot.Name](declare("first", "x", one), declare("second", "x", one)), bot.CodeDuplicateVariable, "second-ident"},
		{"builtin conflict", testutil.MainProgram[bot.Name](declare("var", "add", one)), bot.CodeFunctionNameConflict, "var-ident"},
		{"get_back conflict", testutil.MainProgram[bot.Name](declare("var", "get_back", one)), bot.CodeFunctionNameConflict, "var-ident"},
		{"user function conflict", testutil.MainProgram[bot.Name](declare("var", "main", one)), bot.CodeFunctionNameConflict, "var-ident"},
		{"empty variable", testutil.MainProgram[bot.Name](declare("var", "", one)), bot.CodeInvalidName, "var-ident"},
		{"undefined function", testutil.MainProgram[bot.Name](testutil.Call[bot.Name]("call", "absent")), bot.CodeUndefinedFunction, "call"},
		{"removed get", testutil.MainProgram[bot.Name](testutil.Call[bot.Name]("call", "get")), bot.CodeUndefinedFunction, "call"},
		{"removed set", testutil.MainProgram[bot.Name](testutil.Call[bot.Name]("call", "set")), bot.CodeUndefinedFunction, "call"},
		{"nil statement", testutil.MainProgram[bot.Name](nil), bot.CodeInvalidAST, "body"},
		{"nil pointer statement", testutil.MainProgram[bot.Name]((*bot.Return[bot.Name, bot.Untyped])(nil)), bot.CodeInvalidAST, "body"},
		{"nil expression", testutil.MainProgram[bot.Name](bot.Return[bot.Name, bot.Untyped]{Source: testutil.Source("return")}), bot.CodeInvalidAST, "return"},
		{"nil pointer expression", testutil.MainProgram[bot.Name]((*bot.LiteralExpr[bot.Name, bot.Untyped])(nil)), bot.CodeInvalidAST, "body"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := index.Index(tt.program)
			if result.Functions != nil {
				t.Error("failed indexing exposed a partial program")
			}
			ds := testutil.Diagnostics(t, err, bot.IndexStage)
			d := testutil.FindDiagnostic(t, ds, tt.code, "main", tt.hint)
			if tt.name == "duplicate" && d != nil && d.RelatedHintID != "first-ident" {
				t.Errorf("previous declaration hint = %q, want first-ident", d.RelatedHintID)
			}
		})
	}
}

func TestIndexScopeErrors(t *testing.T) {
	t.Parallel()
	declare := func(hint string) bot.Declaration[bot.Name, bot.Untyped] {
		return bot.Declaration[bot.Name, bot.Untyped]{Type: bot.IntegerType{}, Ident: testutil.Ident(hint, bot.Name("x")), Value: testutil.Literal[bot.Name]("one", bot.IntegerLiteral{Value: 1})}
	}
	program := testutil.MainProgram[bot.Name](
		bot.If[bot.Name, bot.Untyped]{Condition: testutil.Literal[bot.Name]("condition", bot.BoolLiteral{}),
			Then: bot.Block[bot.Name, bot.Untyped]{Stmts: []bot.Stmt[bot.Name, bot.Untyped]{declare("then-x")}},
			Else: &bot.Block[bot.Name, bot.Untyped]{Stmts: []bot.Stmt[bot.Name, bot.Untyped]{testutil.Reference("else-leak", bot.Name("x"))}},
		},
		testutil.Reference("branch-leak", bot.Name("x")),
		declare("outer-x"),
		bot.While[bot.Name, bot.Untyped]{Condition: testutil.Literal[bot.Name]("true", bot.BoolLiteral{Value: true}), Body: bot.Block[bot.Name, bot.Untyped]{Stmts: []bot.Stmt[bot.Name, bot.Untyped]{declare("shadow")}}},
		bot.For[bot.Name, bot.Untyped]{Ident: testutil.Ident("for-var", bot.Name("i")), Start: testutil.Reference("start-leak", bot.Name("i")), End: testutil.Reference("end-leak", bot.Name("i")),
			Body: bot.Block[bot.Name, bot.Untyped]{Stmts: []bot.Stmt[bot.Name, bot.Untyped]{testutil.Reference("loop-use", bot.Name("i"))}},
		},
		testutil.Reference("loop-leak", bot.Name("i")),
	)
	_, err := index.Index(program)
	ds := testutil.Diagnostics(t, err, bot.IndexStage)
	if len(ds) != 6 {
		t.Errorf("expected 6 independent diagnostics, got %d:\n%v", len(ds), ds)
	}
	for _, hint := range []bot.HintID{"else-leak", "branch-leak", "start-leak", "end-leak", "loop-leak"} {
		testutil.RequireDiagnostic(t, ds, bot.CodeUndefinedVariable, "main", hint)
	}
	testutil.RequireDiagnostic(t, ds, bot.CodeDuplicateVariable, "main", "shadow")
}

func TestIndexParametersAndGlobalNames(t *testing.T) {
	t.Parallel()
	program := testutil.MainProgram[bot.Name](testutil.UnitReturn[bot.Name]())
	program.Functions["helper"] = bot.Function[bot.Name, bot.Untyped]{Source: testutil.Source("helper"), Params: []bot.Param[bot.Name]{
		{Ident: testutil.Ident("p1", bot.Name("p")), Type: bot.BoolType{}},
		{Ident: testutil.Ident("p2", bot.Name("p")), Type: bot.BoolType{}},
		{Ident: testutil.Ident("global", bot.Name("main")), Type: bot.UnitType{}},
	}, Body: bot.Block[bot.Name, bot.Untyped]{Stmts: []bot.Stmt[bot.Name, bot.Untyped]{
		bot.Declaration[bot.Name, bot.Untyped]{Ident: testutil.Ident("shadow-param", bot.Name("p")), Value: testutil.Literal[bot.Name]("false", bot.BoolLiteral{})},
	}}}
	program.Functions["add"] = bot.Function[bot.Name, bot.Untyped]{Source: testutil.Source("add-definition")}
	program.Functions["get_back"] = bot.Function[bot.Name, bot.Untyped]{Source: testutil.Source("get-back-definition")}
	program.Functions[""] = bot.Function[bot.Name, bot.Untyped]{Source: testutil.Source("empty-definition")}
	_, err := index.Index(program)
	ds := testutil.Diagnostics(t, err, bot.IndexStage)
	testutil.RequireDiagnostic(t, ds, bot.CodeFunctionConflict, "add", "add-definition")
	testutil.RequireDiagnostic(t, ds, bot.CodeFunctionConflict, "get_back", "get-back-definition")
	testutil.RequireDiagnostic(t, ds, bot.CodeInvalidName, "", "empty-definition")
	testutil.RequireDiagnostic(t, ds, bot.CodeDuplicateVariable, "helper", "p2")
	testutil.RequireDiagnostic(t, ds, bot.CodeFunctionNameConflict, "helper", "global")
	testutil.RequireDiagnostic(t, ds, bot.CodeDuplicateVariable, "helper", "shadow-param")
}

func TestRemovedVectorBuiltinNamesAreAvailable(t *testing.T) {
	t.Parallel()
	for _, name := range []bot.Name{"get", "set"} {
		t.Run(string(name)+"/function", func(t *testing.T) {
			t.Parallel()
			program := testutil.MainProgram[bot.Name](
				testutil.Call[bot.Name]("call", name),
				testutil.UnitReturn[bot.Name](),
			)
			program.Functions[name] = bot.Function[bot.Name, bot.Untyped]{
				Source:     testutil.Source("definition"),
				ReturnType: bot.UnitType{},
				Body: bot.Block[bot.Name, bot.Untyped]{Stmts: []bot.Stmt[bot.Name, bot.Untyped]{
					testutil.UnitReturn[bot.Name](),
				}},
			}
			testutil.Typed(t, testutil.Indexed(t, program))
		})
		t.Run(string(name)+"/variable", func(t *testing.T) {
			t.Parallel()
			program := testutil.MainProgram[bot.Name](
				bot.Declaration[bot.Name, bot.Untyped]{
					Type:  bot.IntegerType{},
					Ident: testutil.Ident("variable", name),
					Value: testutil.Literal[bot.Name]("value", bot.IntegerLiteral{Value: 1}),
				},
				testutil.Reference("read", name),
				testutil.UnitReturn[bot.Name](),
			)
			testutil.Typed(t, testutil.Indexed(t, program))
		})
	}
}

func TestIndexAcceptsPointerNodes(t *testing.T) {
	t.Parallel()
	value := testutil.Literal[bot.Name]("unit", bot.UnitLiteral{})
	result := testutil.Indexed(t, testutil.MainProgram[bot.Name](&bot.Return[bot.Name, bot.Untyped]{Source: testutil.Source("return"), Value: &value}))
	testutil.Typed(t, result)
}
