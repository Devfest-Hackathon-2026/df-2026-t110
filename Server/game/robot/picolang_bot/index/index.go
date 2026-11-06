// Package index resolves picolang variable names to function-local stack slots.
package index

import bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"

// Index resolves variable names to stack slots and checks function/variable
// names. Inputs are not modified. On failure it returns a zero program and
// Diagnostics, collecting independent errors instead of returning a partial AST.
// Source syntax, duplicate function keys, and hint validation belong to parsing.
func Index(program bot.ParsedProgram) (bot.IndexedProgram, error) {
	r := &bot.Reporter{Stage: bot.IndexStage}
	functions := make(map[bot.Name]bot.Source)
	for name := range bot.BuiltinSignatures() {
		functions[name] = bot.Source{}
	}
	names := bot.FunctionNames(program)
	for _, name := range names {
		fn := program.Functions[name]
		r.Function = name
		if name == "" {
			r.Report(fn.Source, bot.CodeInvalidName, "function name must not be empty")
		}
		if _, exists := functions[name]; exists {
			r.Report(fn.Source, bot.CodeFunctionConflict, "function %q conflicts with a builtin", name)
		}
		functions[name] = fn.Source
	}
	result := bot.IndexedProgram{Functions: make(map[bot.Name]bot.Function[bot.VariableIndex, bot.Untyped], len(names))}
	for _, name := range names {
		fn := program.Functions[name]
		r.Function = name
		ix := indexer{Reporter: r, functions: functions, bindings: make(map[bot.Name]namedBinding)}
		params := make([]bot.Param[bot.VariableIndex], len(fn.Params))
		for i, param := range fn.Params {
			params[i] = bot.Param[bot.VariableIndex]{Source: param.Source, Ident: ix.declare(param.Ident), Type: param.Type}
		}
		result.Functions[name] = bot.Function[bot.VariableIndex, bot.Untyped]{Source: fn.Source, Params: params, ReturnType: fn.ReturnType, Body: ix.block(fn.Body)}
	}
	if err := r.Err(); err != nil {
		return bot.IndexedProgram{}, err
	}
	return result, nil
}

type namedBinding struct {
	index  bot.VariableIndex
	source bot.Source
}

type indexer struct {
	*bot.Reporter
	functions map[bot.Name]bot.Source
	bindings  map[bot.Name]namedBinding
	stack     []bot.Name
}

func (ix *indexer) declare(ident bot.Ident[bot.Name]) bot.Ident[bot.VariableIndex] {
	name := ident.Variable
	index := bot.VariableIndex(len(ix.stack))
	switch name {
	case "":
		ix.Report(ident.Source, bot.CodeInvalidName, "variable name must not be empty")
	default:
		if binding, exists := ix.bindings[name]; exists {
			ix.Conflict(ident.Source, bot.CodeDuplicateVariable, binding.source.HintID, "variable %q is already in scope; shadowing is forbidden", name)
		} else if source, exists := ix.functions[name]; exists {
			ix.Conflict(ident.Source, bot.CodeFunctionNameConflict, source.HintID, "variable %q conflicts with a function", name)
		} else {
			ix.bindings[name] = namedBinding{index: index, source: ident.Source}
			ix.stack = append(ix.stack, name)
		}
	}
	return bot.Ident[bot.VariableIndex]{Source: ident.Source, Variable: index}
}

func (ix *indexer) reference(ident bot.Ident[bot.Name]) bot.Ident[bot.VariableIndex] {
	binding, exists := ix.bindings[ident.Variable]
	if !exists {
		ix.Report(ident.Source, bot.CodeUndefinedVariable, "variable %q is not in scope", ident.Variable)
	}
	return bot.Ident[bot.VariableIndex]{Source: ident.Source, Variable: binding.index}
}

func (ix *indexer) leave(base int) {
	for _, name := range ix.stack[base:] {
		delete(ix.bindings, name)
	}
	ix.stack = ix.stack[:base]
}

func (ix *indexer) block(block bot.Block[bot.Name, bot.Untyped]) bot.Block[bot.VariableIndex, bot.Untyped] {
	base := len(ix.stack)
	defer ix.leave(base)
	stmts := make([]bot.Stmt[bot.VariableIndex, bot.Untyped], len(block.Stmts))
	for i, stmt := range block.Stmts {
		stmts[i] = ix.stmt(stmt, block.Source)
	}
	return bot.Block[bot.VariableIndex, bot.Untyped]{Source: block.Source, Stmts: stmts}
}

func (ix *indexer) stmt(stmt bot.Stmt[bot.Name, bot.Untyped], parent bot.Source) bot.Stmt[bot.VariableIndex, bot.Untyped] {
	switch n := bot.NodeValue(stmt).(type) {
	case bot.Expr[bot.Name, bot.Untyped]:
		return ix.expr(n, parent)
	case bot.Declaration[bot.Name, bot.Untyped]:
		// A declaration is not visible to its own initializer.
		value := ix.expr(n.Value, n.Source)
		return bot.Declaration[bot.VariableIndex, bot.Untyped]{Source: n.Source, Type: n.Type, Ident: ix.declare(n.Ident), Value: value}
	case bot.Assignment[bot.Name, bot.Untyped]:
		ident := ix.reference(n.Ident)
		return bot.Assignment[bot.VariableIndex, bot.Untyped]{Source: n.Source, Ident: ident, Value: ix.expr(n.Value, n.Source)}
	case bot.If[bot.Name, bot.Untyped]:
		result := bot.If[bot.VariableIndex, bot.Untyped]{Source: n.Source, Condition: ix.expr(n.Condition, n.Source), Then: ix.block(n.Then)}
		if n.Else != nil {
			other := ix.block(*n.Else)
			result.Else = &other
		}
		return result
	case bot.While[bot.Name, bot.Untyped]:
		return bot.While[bot.VariableIndex, bot.Untyped]{Source: n.Source, Condition: ix.expr(n.Condition, n.Source), Body: ix.block(n.Body)}
	case bot.For[bot.Name, bot.Untyped]:
		start, end := ix.expr(n.Start, n.Source), ix.expr(n.End, n.Source)
		base := len(ix.stack)
		ident := ix.declare(n.Ident)
		body := ix.block(n.Body)
		ix.leave(base)
		return bot.For[bot.VariableIndex, bot.Untyped]{Source: n.Source, Ident: ident, Start: start, End: end, Body: body}
	case bot.Return[bot.Name, bot.Untyped]:
		return bot.Return[bot.VariableIndex, bot.Untyped]{Source: n.Source, Value: ix.expr(n.Value, n.Source)}
	case bot.Break[bot.Name, bot.Untyped]:
		return bot.Break[bot.VariableIndex, bot.Untyped](n)
	case bot.Continue[bot.Name, bot.Untyped]:
		return bot.Continue[bot.VariableIndex, bot.Untyped](n)
	default:
		ix.Report(bot.SourceOf(stmt, parent), bot.CodeInvalidAST, "invalid statement node %T", stmt)
		return nil
	}
}

func (ix *indexer) expr(expr bot.Expr[bot.Name, bot.Untyped], parent bot.Source) bot.Expr[bot.VariableIndex, bot.Untyped] {
	switch n := bot.NodeValue(expr).(type) {
	case bot.LiteralExpr[bot.Name, bot.Untyped]:
		return bot.LiteralExpr[bot.VariableIndex, bot.Untyped]{Source: n.Source, Value: n.Value}
	case bot.IdentExpr[bot.Name, bot.Untyped]:
		return bot.IdentExpr[bot.VariableIndex, bot.Untyped]{Ident: ix.reference(n.Ident)}
	case bot.CallExpr[bot.Name, bot.Untyped]:
		if _, exists := ix.functions[n.Func]; !exists {
			ix.Report(n.Source, bot.CodeUndefinedFunction, "function %q is not defined", n.Func)
		}
		args := make([]bot.Expr[bot.VariableIndex, bot.Untyped], len(n.Args))
		for i, arg := range n.Args {
			args[i] = ix.expr(arg, n.Source)
		}
		return bot.CallExpr[bot.VariableIndex, bot.Untyped]{Source: n.Source, Func: n.Func, Args: args}
	default:
		ix.Report(bot.SourceOf(expr, parent), bot.CodeInvalidAST, "invalid expression node %T", expr)
		return nil
	}
}
