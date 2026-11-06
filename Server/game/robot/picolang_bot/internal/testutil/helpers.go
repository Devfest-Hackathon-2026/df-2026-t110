package testutil

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	indexstage "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/index"
	typecheckstage "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/typecheck"
)

var (
	_ bot.IndexFunc     = indexstage.Index
	_ bot.TypeCheckFunc = typecheckstage.TypeCheck
)

func Source(hint string) bot.Source { return bot.Source{HintID: bot.HintID(hint)} }

func Ident[V bot.Variable](hint string, variable V) bot.Ident[V] {
	return bot.Ident[V]{Source: Source(hint), Variable: variable}
}

func Literal[V bot.Variable](hint string, value bot.Literal) bot.LiteralExpr[V, bot.Untyped] {
	return bot.LiteralExpr[V, bot.Untyped]{Source: Source(hint), Value: value}
}

func Reference[V bot.Variable](hint string, variable V) bot.IdentExpr[V, bot.Untyped] {
	return bot.IdentExpr[V, bot.Untyped]{Ident: Ident(hint, variable)}
}

func Call[V bot.Variable](hint string, name bot.Name, args ...bot.Expr[V, bot.Untyped]) bot.CallExpr[V, bot.Untyped] {
	if args == nil {
		args = []bot.Expr[V, bot.Untyped]{}
	}
	return bot.CallExpr[V, bot.Untyped]{Source: Source(hint), Func: name, Args: args}
}

func UnitReturn[V bot.Variable]() bot.Return[V, bot.Untyped] {
	return bot.Return[V, bot.Untyped]{Source: Source("return"), Value: Literal[V]("unit", bot.UnitLiteral{})}
}

func MainProgram[V bot.Variable](stmts ...bot.Stmt[V, bot.Untyped]) bot.Program[V, bot.Untyped] {
	return bot.Program[V, bot.Untyped]{Functions: map[bot.Name]bot.Function[V, bot.Untyped]{
		"main": {Source: Source("main"), Params: []bot.Param[V]{}, ReturnType: bot.UnitType{}, Body: bot.Block[V, bot.Untyped]{Source: Source("body"), Stmts: stmts}},
	}}
}

func Diagnostics(t *testing.T, err error, stage bot.AnalysisStage) bot.Diagnostics {
	t.Helper()
	var ds bot.Diagnostics
	if !errors.As(err, &ds) || len(ds) == 0 {
		t.Fatalf("expected nonempty Diagnostics, got %v", err)
	}
	var first *bot.Diagnostic
	if !errors.As(err, &first) || first != ds[0] {
		t.Errorf("errors.As did not expose the first diagnostic: %v", err)
	}
	for _, d := range ds {
		if d.Stage != stage || d.Code == "" || d.Message == "" {
			t.Errorf("incomplete diagnostic: %+v", d)
		}
		for _, part := range []string{string(d.Stage), string(d.Code), string(d.Function), string(d.HintID)} {
			if !strings.Contains(d.Error(), part) {
				t.Errorf("message %q omits %q", d.Error(), part)
			}
		}
	}
	return ds
}

func FindDiagnostic(t *testing.T, ds bot.Diagnostics, code bot.DiagnosticCode, function bot.Name, hint bot.HintID) *bot.Diagnostic {
	t.Helper()
	for _, d := range ds {
		if d.Code == code && d.Function == function && d.HintID == hint {
			return d
		}
	}
	t.Errorf("missing diagnostic %s in %q at %q; got:\n%v", code, function, hint, ds)
	return nil
}

func RequireDiagnostic(t *testing.T, ds bot.Diagnostics, code bot.DiagnosticCode, function bot.Name, hint bot.HintID) {
	t.Helper()
	_ = FindDiagnostic(t, ds, code, function, hint)
}

func Typed(t *testing.T, program bot.IndexedProgram) bot.TypedProgram {
	t.Helper()
	result, err := typecheckstage.TypeCheck(program)
	if err != nil {
		t.Fatalf("type checking fixture: %v", err)
	}
	return result
}

func Indexed(t *testing.T, program bot.ParsedProgram) bot.IndexedProgram {
	t.Helper()
	result, err := indexstage.Index(program)
	if err != nil {
		t.Fatalf("indexing fixture: %v", err)
	}
	return result
}

func AssertType(t *testing.T, actual, expected bot.Type) {
	t.Helper()
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("type = %#v, want %#v", actual, expected)
	}
}
