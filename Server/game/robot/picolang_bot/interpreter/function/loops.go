package function

import (
	"errors"

	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
)

type loop interface {
	task
	breakBase() int
	continueBase() int
}

// Pending loop continuations mark the nearest enclosing loop. Unwinding drops
// block locals before removing their cleanup continuations.
func (m *Machine) unwind(continuing bool) {
	for i := len(m.tasks) - 1; i >= 0; i-- {
		if boundary, ok := m.tasks[i].(loop); ok {
			end, base := i, boundary.breakBase()
			if continuing {
				end, base = i+1, boundary.continueBase()
			}
			m.leave(base)
			clear(m.tasks[end:])
			m.tasks = m.tasks[:end]
			return
		}
	}
	panic(errors.New("loop control outside a loop"))
}

type whileCondition struct {
	node bot.While[bot.VariableIndex, bot.Type]
	base int
}

func (t whileCondition) run(m *Machine) {
	m.push(whileDecision{loop: t})
	m.push(evaluate{expr: t.node.Condition})
}

type whileDecision struct{ loop whileCondition }

func (t whileDecision) run(m *Machine) {
	id := m.pop()
	condition := m.value(id).(memory.Bool).Value
	m.drop(id)
	if condition {
		m.push(whileNext(t.loop))
		m.push(enterBlock{block: t.loop.node.Body})
	}
}

type whileNext whileCondition

func (t whileNext) breakBase() int    { return t.base }
func (t whileNext) continueBase() int { return t.base }
func (t whileNext) run(m *Machine)    { m.push(whileCondition(t)) }

type forBounds struct {
	node bot.For[bot.VariableIndex, bot.Type]
	base int
}

func (t forBounds) run(m *Machine) {
	endID, startID := m.pop(), m.pop()
	start := m.value(startID).(memory.Integer).Value
	end := m.value(endID).(memory.Integer).Value
	m.drop(startID)
	m.drop(endID)
	if start < end {
		m.define(t.node.Ident, m.allocate(memory.Integer{Value: start}))
		m.push(forNext{node: t.node, base: t.base, current: start, end: end})
		m.push(enterBlock{block: t.node.Body})
	}
}

type forNext struct {
	node    bot.For[bot.VariableIndex, bot.Type]
	base    int
	current int64
	end     int64
}

func (t forNext) breakBase() int    { return t.base }
func (t forNext) continueBase() int { return t.base + 1 }

func (t forNext) run(m *Machine) {
	// Every executed iteration has current < end, so current cannot be MaxInt64.
	t.current++
	if t.current >= t.end {
		m.leave(t.base)
		return
	}
	m.current(t.node.Source)
	m.assign(t.node.Ident.Variable, m.allocate(memory.Integer{Value: t.current}))
	m.push(t)
	m.push(enterBlock{block: t.node.Body})
}
