package function

import (
	"errors"
	"fmt"
	"slices"

	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
)

type parameter struct{ index int }

func (t parameter) run(m *Machine) {
	variable := &m.variables[t.index]
	m.current(bot.Source{HintID: variable.hint})
	m.resources.Time.Increment()
	if m.resources.Events != nil {
		m.resources.Events.Add(robotevent.VariableInitEvent{HintID: variable.hint, ValueID: variable.value})
		variable.announced = true
	}
}

type enterBlock struct {
	block bot.Block[bot.VariableIndex, bot.Type]
}

func (t enterBlock) run(m *Machine) {
	m.current(t.block.Source)
	m.resources.Time.Increment()
	m.push(leaveScope{base: len(m.variables)})
	m.push(blockNext{stmts: t.block.Stmts})
}

type blockNext struct {
	stmts []bot.Stmt[bot.VariableIndex, bot.Type]
}

func (t blockNext) run(m *Machine) {
	if len(t.stmts) != 0 {
		m.push(blockNext{stmts: t.stmts[1:]})
		m.push(statement{stmt: t.stmts[0]})
	}
}

type leaveScope struct{ base int }

func (t leaveScope) run(m *Machine) { m.leave(t.base) }

type statement struct {
	stmt bot.Stmt[bot.VariableIndex, bot.Type]
}

func (t statement) run(m *Machine) {
	if expr, ok := t.stmt.(bot.Expr[bot.VariableIndex, bot.Type]); ok {
		m.push(discard{})
		m.push(evaluate{expr: expr})
		return
	}
	m.current(bot.SourceOf(t.stmt, m.source))
	switch n := bot.NodeValue(t.stmt).(type) {
	case bot.Declaration[bot.VariableIndex, bot.Type]:
		m.push(storeVariable{ident: n.Ident, declaration: true})
		m.push(evaluate{expr: n.Value})
	case bot.Assignment[bot.VariableIndex, bot.Type]:
		m.push(storeVariable{ident: n.Ident})
		m.push(evaluate{expr: n.Value})
	case bot.If[bot.VariableIndex, bot.Type]:
		m.push(branch{node: n})
		m.push(evaluate{expr: n.Condition})
	case bot.While[bot.VariableIndex, bot.Type]:
		m.push(whileCondition{node: n, base: len(m.variables)})
	case bot.For[bot.VariableIndex, bot.Type]:
		m.push(forBounds{node: n, base: len(m.variables)})
		m.push(evaluate{expr: n.End})
		m.push(evaluate{expr: n.Start})
	case bot.Return[bot.VariableIndex, bot.Type]:
		m.push(returnValue{})
		m.push(evaluate{expr: n.Value})
	case bot.Break[bot.VariableIndex, bot.Type]:
		m.unwind(false)
	case bot.Continue[bot.VariableIndex, bot.Type]:
		m.unwind(true)
	default:
		panic(fmt.Errorf("invalid statement %T", t.stmt))
	}
}

type evaluate struct {
	expr bot.Expr[bot.VariableIndex, bot.Type]
}

func (t evaluate) run(m *Machine) {
	m.current(bot.SourceOf(t.expr, m.source))
	switch n := bot.NodeValue(t.expr).(type) {
	case bot.LiteralExpr[bot.VariableIndex, bot.Type]:
		var value memory.Value
		switch literal := bot.NodeValue(n.Value).(type) {
		case bot.BoolLiteral:
			value = memory.Bool{Value: literal.Value}
		case bot.IntegerLiteral:
			value = memory.Integer{Value: literal.Value}
		case bot.UnitLiteral:
			value = memory.Unit{}
		case bot.RotationLiteral:
			value = memory.Rotation{Value: literal.Value}
		case bot.DistanceTargetLiteral:
			value = memory.DistanceTarget{Value: literal.Value}
		default:
			panic(fmt.Errorf("invalid literal %T", n.Value))
		}
		m.operands = append(m.operands, m.allocate(value))
	case bot.IdentExpr[bot.VariableIndex, bot.Type]:
		m.resources.Time.Increment()
		m.operands = append(m.operands, m.retain(m.variables[n.Variable].value))
	case bot.CallExpr[bot.VariableIndex, bot.Type]:
		m.push(invoke{node: n})
		for i := len(n.Args) - 1; i >= 0; i-- {
			m.push(evaluate{expr: n.Args[i]})
		}
	default:
		panic(fmt.Errorf("invalid expression %T", t.expr))
	}
}

type invoke struct {
	node bot.CallExpr[bot.VariableIndex, bot.Type]
}

func (t invoke) run(m *Machine) {
	m.current(t.node.Source)
	m.resources.Time.Increment()
	base := len(m.operands) - len(t.node.Args)
	m.waiting = Call{Source: t.node.Source, Func: t.node.Func, Args: slices.Clone(m.operands[base:]), ResultType: t.node.Annotation}
	m.operands = m.operands[:base]
	m.phase = awaiting
}

type discard struct{}

func (discard) run(m *Machine) { m.drop(m.pop()) }

type storeVariable struct {
	ident       bot.Ident[bot.VariableIndex]
	declaration bool
}

func (t storeVariable) run(m *Machine) {
	m.current(t.ident.Source)
	if t.declaration {
		m.define(t.ident, m.pop())
	} else {
		m.assign(t.ident.Variable, m.pop())
	}
}

type branch struct {
	node bot.If[bot.VariableIndex, bot.Type]
}

func (t branch) run(m *Machine) {
	id := m.pop()
	condition := m.value(id).(memory.Bool).Value
	m.drop(id)
	if condition {
		m.push(enterBlock{block: t.node.Then})
	} else if t.node.Else != nil {
		m.push(enterBlock{block: *t.node.Else})
	}
}

type returnValue struct{}

func (returnValue) run(m *Machine) {
	id := m.pop()
	m.leave(0)
	m.forget(id)
	if err := m.Close(); err != nil {
		panic(errors.Join(err, m.resources.Memory.Release(id)))
	}
	m.result = id
	m.phase = finished
}
