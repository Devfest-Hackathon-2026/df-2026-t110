// Package typecheck validates indexed picolang programs and annotates expressions.
package typecheck

import (
	"fmt"
	"strconv"
	"strings"

	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
)

// TypeCheck checks signatures, stack slots, expressions, and control flow, then
// annotates every expression with a concrete language type. Stack builtin type
// parameters are inferred from arguments and surrounding declared types.
// Inputs are not modified; failures return a zero program and Diagnostics.
func TypeCheck(program bot.IndexedProgram) (bot.TypedProgram, error) {
	r := &bot.Reporter{Stage: bot.TypeCheckStage}
	signatures := bot.BuiltinSignatures()
	names := bot.FunctionNames(program)
	for _, name := range names {
		r.Function = name
		fn := program.Functions[name]
		if name == "" {
			r.Report(fn.Source, bot.CodeInvalidName, "function name must not be empty")
		}
		if _, builtin := signatures[name]; builtin {
			r.Report(fn.Source, bot.CodeFunctionConflict, "function %q conflicts with a builtin", name)
		}
		sig := bot.Signature{Params: make([]bot.Type, len(fn.Params)), Result: checkedDeclaredType(r, fn.Source, fn.ReturnType)}
		for i, param := range fn.Params {
			sig.Params[i] = checkedDeclaredType(r, param.Source, param.Type)
		}
		signatures[name] = sig
	}
	r.Function = "main"
	main, exists := program.Functions["main"]
	if !exists {
		r.Report(bot.Source{}, bot.CodeInvalidMain, "entry point main is missing; expected main: () -> unit")
	} else if sig := signatures["main"]; len(sig.Params) != 0 || !isUnit(sig.Result) {
		r.Mismatch(main.Source, bot.CodeInvalidMain, "() -> unit", signatureName(sig), "entry point main must have signature () -> unit, got %s", signatureName(sig))
	}
	result := bot.TypedProgram{Functions: make(map[bot.Name]bot.Function[bot.VariableIndex, bot.Type], len(names))}
	for _, name := range names {
		r.Function = name
		fn, sig := program.Functions[name], signatures[name]
		c := typeChecker{Reporter: r, signatures: signatures, returnType: sig.Result}
		params := make([]bot.Param[bot.VariableIndex], len(fn.Params))
		for i, param := range fn.Params {
			c.declare(param.Ident, sig.Params[i])
			params[i] = bot.Param[bot.VariableIndex]{Source: param.Source, Ident: param.Ident, Type: sig.Params[i]}
		}
		body := c.block(fn.Body)
		if !endsWithReturn(fn.Body) {
			r.Report(fn.Body.Source, bot.CodeMissingReturn, "function body must end with an explicit return statement")
		}
		c.finish()
		result.Functions[name] = bot.Function[bot.VariableIndex, bot.Type]{Source: fn.Source, Params: params, ReturnType: sig.Result, Body: body}
	}
	if err := r.Err(); err != nil {
		return bot.TypedProgram{}, err
	}
	return result, nil
}

func checkedDeclaredType(r *bot.Reporter, source bot.Source, t bot.Type) bot.Type {
	result, valid := bot.CanonicalType(t)
	if !valid {
		r.Report(source, bot.CodeInvalidType, "invalid declared type %T; expected a primitive type or stack with a valid element type", t)
	}
	return result
}

func isUnit(t bot.Type) bool {
	_, ok := t.(bot.UnitType)
	// An invalid return type already has its own diagnostic.
	return ok || bot.IsInvalidType(t)
}

func signatureName(sig bot.Signature) string {
	params := make([]string, len(sig.Params))
	for i, t := range sig.Params {
		params[i] = bot.TypeName(t)
	}
	return "(" + strings.Join(params, ", ") + ") -> " + bot.TypeName(sig.Result)
}

func endsWithReturn(block bot.Block[bot.VariableIndex, bot.Untyped]) bool {
	if len(block.Stmts) == 0 {
		return false
	}
	_, ok := bot.NodeValue(block.Stmts[len(block.Stmts)-1]).(bot.Return[bot.VariableIndex, bot.Untyped])
	return ok
}

type typeChecker struct {
	*bot.Reporter
	signatures  map[bot.Name]bot.Signature
	returnType  bot.Type
	stack       []bot.Type
	loopDepth   int
	inference   bot.Inference
	annotations []*bot.Type
}

func (c *typeChecker) declare(ident bot.Ident[bot.VariableIndex], t bot.Type) {
	expected := bot.VariableIndex(len(c.stack))
	if ident.Variable != expected {
		c.Mismatch(ident.Source, bot.CodeInvalidVariableIndex, fmt.Sprint(expected), fmt.Sprint(ident.Variable), "declaration has slot %d; expected next stack slot %d", ident.Variable, expected)
	}
	c.stack = append(c.stack, t)
}

func (c *typeChecker) reference(ident bot.Ident[bot.VariableIndex]) bot.Type {
	if ident.Variable >= bot.VariableIndex(len(c.stack)) {
		c.Report(ident.Source, bot.CodeInvalidVariableIndex, "variable slot %d is not in scope (active slots: %d)", ident.Variable, len(c.stack))
		return bot.InvalidType()
	}
	return c.stack[ident.Variable]
}

func (c *typeChecker) expect(source bot.Source, context string, expected, actual bot.Type) {
	want, got := bot.TypeName(expected), bot.TypeName(actual)
	if !bot.UnifyTypes(expected, actual) {
		c.Mismatch(source, bot.CodeTypeMismatch, want, got, "%s: expected %s, got %s", context, want, got)
	}
}

func (c *typeChecker) block(block bot.Block[bot.VariableIndex, bot.Untyped]) bot.Block[bot.VariableIndex, bot.Type] {
	base := len(c.stack)
	defer func() { c.stack = c.stack[:base] }()
	stmts := make([]bot.Stmt[bot.VariableIndex, bot.Type], len(block.Stmts))
	for i, stmt := range block.Stmts {
		stmts[i] = c.stmt(stmt, block.Source)
	}
	return bot.Block[bot.VariableIndex, bot.Type]{Source: block.Source, Stmts: stmts}
}

func (c *typeChecker) stmt(stmt bot.Stmt[bot.VariableIndex, bot.Untyped], parent bot.Source) bot.Stmt[bot.VariableIndex, bot.Type] {
	switch n := bot.NodeValue(stmt).(type) {
	case bot.Expr[bot.VariableIndex, bot.Untyped]:
		expr, _ := c.expr(n, parent)
		return expr
	case bot.Declaration[bot.VariableIndex, bot.Untyped]:
		t := checkedDeclaredType(c.Reporter, n.Source, n.Type)
		value, actual := c.expr(n.Value, n.Source)
		c.expect(bot.SourceOf(n.Value, n.Source), "variable initializer", t, actual)
		c.declare(n.Ident, t)
		return bot.Declaration[bot.VariableIndex, bot.Type]{Source: n.Source, Type: t, Ident: n.Ident, Value: value}
	case bot.Assignment[bot.VariableIndex, bot.Untyped]:
		t := c.reference(n.Ident)
		value, actual := c.expr(n.Value, n.Source)
		c.expect(bot.SourceOf(n.Value, n.Source), "assignment", t, actual)
		return bot.Assignment[bot.VariableIndex, bot.Type]{Source: n.Source, Ident: n.Ident, Value: value}
	case bot.If[bot.VariableIndex, bot.Untyped]:
		condition, actual := c.expr(n.Condition, n.Source)
		c.expect(bot.SourceOf(n.Condition, n.Source), "if condition", bot.BoolType{}, actual)
		result := bot.If[bot.VariableIndex, bot.Type]{Source: n.Source, Condition: condition, Then: c.block(n.Then)}
		if n.Else != nil {
			other := c.block(*n.Else)
			result.Else = &other
		}
		return result
	case bot.While[bot.VariableIndex, bot.Untyped]:
		condition, actual := c.expr(n.Condition, n.Source)
		c.expect(bot.SourceOf(n.Condition, n.Source), "while condition", bot.BoolType{}, actual)
		c.loopDepth++
		body := c.block(n.Body)
		c.loopDepth--
		return bot.While[bot.VariableIndex, bot.Type]{Source: n.Source, Condition: condition, Body: body}
	case bot.For[bot.VariableIndex, bot.Untyped]:
		start, startType := c.expr(n.Start, n.Source)
		c.expect(bot.SourceOf(n.Start, n.Source), "for start", bot.IntegerType{}, startType)
		end, endType := c.expr(n.End, n.Source)
		c.expect(bot.SourceOf(n.End, n.Source), "for end", bot.IntegerType{}, endType)
		base := len(c.stack)
		c.declare(n.Ident, bot.IntegerType{})
		c.loopDepth++
		body := c.block(n.Body)
		c.loopDepth--
		c.stack = c.stack[:base]
		return bot.For[bot.VariableIndex, bot.Type]{Source: n.Source, Ident: n.Ident, Start: start, End: end, Body: body}
	case bot.Return[bot.VariableIndex, bot.Untyped]:
		value, actual := c.expr(n.Value, n.Source)
		c.expect(bot.SourceOf(n.Value, n.Source), "return value", c.returnType, actual)
		return bot.Return[bot.VariableIndex, bot.Type]{Source: n.Source, Value: value}
	case bot.Break[bot.VariableIndex, bot.Untyped]:
		if c.loopDepth == 0 {
			c.Report(n.Source, bot.CodeInvalidLoopControl, "break is only allowed inside a loop")
		}
		return bot.Break[bot.VariableIndex, bot.Type](n)
	case bot.Continue[bot.VariableIndex, bot.Untyped]:
		if c.loopDepth == 0 {
			c.Report(n.Source, bot.CodeInvalidLoopControl, "continue is only allowed inside a loop")
		}
		return bot.Continue[bot.VariableIndex, bot.Type](n)
	default:
		c.Report(bot.SourceOf(stmt, parent), bot.CodeInvalidAST, "invalid statement node %T", stmt)
		return nil
	}
}

// Expressions are allocated as nodes so annotation fields can be finalized
// after constraints from later arguments or enclosing expressions are known.
func (c *typeChecker) expr(expr bot.Expr[bot.VariableIndex, bot.Untyped], parent bot.Source) (bot.Expr[bot.VariableIndex, bot.Type], bot.Type) {
	switch n := bot.NodeValue(expr).(type) {
	case bot.LiteralExpr[bot.VariableIndex, bot.Untyped]:
		value := bot.NodeValue(n.Value)
		t := c.literalType(value, n.Source)
		result := &bot.LiteralExpr[bot.VariableIndex, bot.Type]{Source: n.Source, Annotation: t, Value: value}
		c.annotations = append(c.annotations, &result.Annotation)
		return result, t
	case bot.IdentExpr[bot.VariableIndex, bot.Untyped]:
		t := c.reference(n.Ident)
		result := &bot.IdentExpr[bot.VariableIndex, bot.Type]{Ident: n.Ident, Annotation: t}
		c.annotations = append(c.annotations, &result.Annotation)
		return result, t
	case bot.CallExpr[bot.VariableIndex, bot.Untyped]:
		return c.call(n)
	default:
		c.Report(bot.SourceOf(expr, parent), bot.CodeInvalidAST, "invalid expression node %T", expr)
		return nil, bot.InvalidType()
	}
}

func (c *typeChecker) literalType(value bot.Literal, source bot.Source) bot.Type {
	switch value := value.(type) {
	case bot.BoolLiteral:
		return bot.BoolType{}
	case bot.IntegerLiteral:
		return bot.IntegerType{}
	case bot.UnitLiteral:
		return bot.UnitType{}
	case bot.RotationLiteral:
		if value.Value == values.Left || value.Value == values.Right {
			return bot.RotationType{}
		}
		c.Report(source, bot.CodeInvalidAST, "invalid rotation literal %q; expected left or right", value.Value)
	case bot.DistanceTargetLiteral:
		if value.Value.Valid() {
			return bot.DistanceTargetType{}
		}
		c.Report(source, bot.CodeInvalidAST, "invalid distance target literal %q", value.Value)
	default:
		c.Report(source, bot.CodeInvalidAST, "invalid literal %T", value)
	}
	return bot.InvalidType()
}

func (c *typeChecker) call(n bot.CallExpr[bot.VariableIndex, bot.Untyped]) (bot.Expr[bot.VariableIndex, bot.Type], bot.Type) {
	args := make([]bot.Expr[bot.VariableIndex, bot.Type], len(n.Args))
	argTypes := make([]bot.Type, len(n.Args))
	for i, arg := range n.Args {
		args[i], argTypes[i] = c.expr(arg, n.Source)
	}
	sig, exists := c.signatures[n.Func]
	resultType := bot.InvalidType()
	if !exists {
		c.Report(n.Source, bot.CodeUndefinedFunction, "function %q is not defined", n.Func)
		for _, argType := range argTypes {
			bot.UnifyTypes(bot.InvalidType(), argType)
		}
	} else {
		instantiated := c.inference.Instantiate(sig, n.Source)
		params := instantiated.Params
		resultType = instantiated.Result
		if len(args) != len(params) {
			c.Mismatch(n.Source, bot.CodeArgumentCount, strconv.Itoa(len(params)), strconv.Itoa(len(args)), "call to %q expects %d arguments, got %d", n.Func, len(params), len(args))
			bot.UnifyTypes(bot.InvalidType(), resultType)
			for _, param := range params {
				bot.UnifyTypes(bot.InvalidType(), param)
			}
			for _, argType := range argTypes {
				bot.UnifyTypes(bot.InvalidType(), argType)
			}
		}
		for i := 0; i < len(args) && i < len(params); i++ {
			c.expect(bot.SourceOf(n.Args[i], n.Source), fmt.Sprintf("argument %d of %q", i+1, n.Func), params[i], argTypes[i])
		}
	}
	result := &bot.CallExpr[bot.VariableIndex, bot.Type]{Source: n.Source, Annotation: resultType, Func: n.Func, Args: args}
	c.annotations = append(c.annotations, &result.Annotation)
	return result, resultType
}

func (c *typeChecker) finish() {
	for _, source := range c.inference.Unresolved() {
		c.Report(source, bot.CodeCannotInferType, "cannot infer stack element type; provide a declared type or a constraining argument")
	}
	for _, annotation := range c.annotations {
		*annotation = bot.ResolveType(*annotation)
	}
}
