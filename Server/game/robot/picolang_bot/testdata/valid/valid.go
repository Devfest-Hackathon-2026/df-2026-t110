package valid

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
)

// All source constructs are available in every stage, with one consistent
// variable representation and expression annotation throughout each tree.
func function[V bot.Variable, A bot.Annotation](variable V, annotation A) bot.Function[V, A] {
	ident := bot.Ident[V]{Source: bot.Source{HintID: "ident-hint"}, Variable: variable}
	literal := bot.LiteralExpr[V, A]{Annotation: annotation, Value: bot.BoolLiteral{Value: true}}
	reference := bot.IdentExpr[V, A]{Ident: ident, Annotation: annotation}
	call := bot.CallExpr[V, A]{
		Annotation: annotation,
		Func:       "not",
		Args:       []bot.Expr[V, A]{reference},
	}
	body := bot.Block[V, A]{Stmts: []bot.Stmt[V, A]{
		literal, reference, call,
		bot.Declaration[V, A]{Type: bot.BoolType{}, Ident: ident, Value: literal},
		bot.Assignment[V, A]{Ident: ident, Value: call},
		bot.If[V, A]{Condition: literal, Else: &bot.Block[V, A]{}},
		bot.While[V, A]{Condition: literal},
		bot.For[V, A]{Ident: ident,
			Start: bot.LiteralExpr[V, A]{Annotation: annotation, Value: bot.IntegerLiteral{Value: 0}},
			End:   bot.LiteralExpr[V, A]{Annotation: annotation, Value: bot.IntegerLiteral{Value: 2}},
		},
		bot.Break[V, A]{}, bot.Continue[V, A]{},
		bot.Return[V, A]{Value: bot.LiteralExpr[V, A]{Annotation: annotation, Value: bot.UnitLiteral{}}},
	}}
	return bot.Function[V, A]{
		Params:     []bot.Param[V]{{Ident: ident, Type: bot.BoolType{}}},
		ReturnType: bot.UnitType{},
		Body:       body,
	}
}

var (
	_ = bot.ParsedProgram{Functions: map[bot.Name]bot.Function[bot.Name, bot.Untyped]{
		"main": function(bot.Name("x"), bot.Untyped{}),
	}}
	_ = bot.IndexedProgram{Functions: map[bot.Name]bot.Function[bot.VariableIndex, bot.Untyped]{
		"main": function(bot.VariableIndex(0), bot.Untyped{}),
	}}
	_ = bot.TypedProgram{Functions: map[bot.Name]bot.Function[bot.VariableIndex, bot.Type]{
		"main": function[bot.VariableIndex, bot.Type](0, bot.BoolType{}),
	}}
	_ bot.CompoundType = bot.VectorType{ElementType: bot.VectorType{ElementType: bot.IntegerType{}}}
	_                  = []bot.PrimitiveType{bot.BoolType{}, bot.IntegerType{}, bot.UnitType{}, bot.TerrainType{}, bot.RotationType{}}
	_                  = []bot.Literal{bot.BoolLiteral{}, bot.IntegerLiteral{}, bot.UnitLiteral{}, bot.RotationLiteral{Value: game.Left}}
)

func pipeline(parse bot.ParseFunc, index bot.IndexFunc, check bot.TypeCheckFunc, source []byte) (bot.TypedProgram, error) {
	parsed, err := parse(source)
	if err != nil {
		return bot.TypedProgram{}, err
	}
	indexed, err := index(parsed)
	if err != nil {
		return bot.TypedProgram{}, err
	}
	return check(indexed)
}
