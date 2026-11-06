package invalid

import bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"

var (
	parsed  bot.ParsedProgram
	indexed bot.IndexedProgram
	typed   bot.TypedProgram
	name    bot.Name
	slot    bot.VariableIndex
	check   bot.TypeCheckFunc
	index   bot.IndexFunc

	namedExpr   bot.Expr[bot.Name, bot.Untyped]
	indexedExpr bot.Expr[bot.VariableIndex, bot.Untyped]
	typedExpr   bot.Expr[bot.VariableIndex, bot.Type]

	_ bot.IndexedProgram = parsed  // want-error
	_ bot.TypedProgram   = indexed // want-error
	_ bot.ParsedProgram  = typed   // want-error

	_ bot.Ident[bot.VariableIndex]             = bot.Ident[bot.VariableIndex]{Variable: name} // want-error
	_ bot.Ident[bot.Name]                      = bot.Ident[bot.Name]{Variable: slot}          // want-error
	_ bot.Expr[bot.VariableIndex, bot.Untyped] = namedExpr                                    // want-error
	_ bot.Expr[bot.VariableIndex, bot.Type]    = indexedExpr                                  // want-error
	_ bot.Stmt[bot.VariableIndex, bot.Untyped] = typedExpr                                    // want-error

	_ = bot.CallExpr[bot.VariableIndex, bot.Type]{Args: []bot.Expr[bot.VariableIndex, bot.Type]{indexedExpr}}                   // want-error
	_ = bot.Block[bot.VariableIndex, bot.Type]{Stmts: []bot.Stmt[bot.VariableIndex, bot.Type]{bot.Break[bot.Name, bot.Type]{}}} // want-error
	_ = bot.If[bot.VariableIndex, bot.Type]{Then: bot.Block[bot.VariableIndex, bot.Untyped]{}}                                  // want-error
	_ = bot.LiteralExpr[bot.VariableIndex, bot.Type]{Annotation: bot.Untyped{}}                                                 // want-error
	_ = bot.Declaration[bot.VariableIndex, bot.Type]{Ident: bot.Ident[bot.Name]{}}                                              // want-error
	_ = bot.Function[bot.VariableIndex, bot.Type]{Params: []bot.Param[bot.Name]{}}                                              // want-error

	_, _ = check(parsed) // want-error
	_, _ = index(typed)  // want-error
)

type foreignExpr struct{}

var _ bot.Expr[bot.VariableIndex, bot.Type] = foreignExpr{} // want-error

type arbitraryVariable = bot.Program[string, bot.Untyped] // want-error
