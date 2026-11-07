package function

import (
	"errors"
	"fmt"
	"slices"

	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
)

type task interface{ run(*Machine) }

type binding struct {
	value     memory.ID
	hint      bot.HintID
	announced bool
}

type phase uint8

const (
	executing phase = iota
	awaiting
	finished
	closed
)

// Machine holds one invocation's continuations, operands, and local variables.
// Construct it with New, then Step until Call or Returned. Resume supplies a
// call's result. Close abandons an unfinished invocation and releases its values.
// The checked AST must not be modified while the machine is in use.
type Machine struct {
	resources Resources
	phase     phase
	tasks     []task
	operands  []memory.ID
	variables []binding
	owned     map[memory.ID]uint64
	waiting   Call
	result    memory.ID
	source    bot.Source
}

// Source identifies the most recently executed node, including after completion.
func (m *Machine) Source() bot.Source { return m.source }

// New retains args; the caller keeps its original references. fn must come from
// successful type checking. Argument and resumed result values must match their
// checked types, including vector element types. Only outer kinds are checked
// here so retaining shared vectors remains O(1).
// Call requests charge time at their call site; entry into this machine does not
// charge the function call again. Parameters charge variable-definition time.
func New(fn bot.Function[bot.VariableIndex, bot.Type], args []memory.ID, resources Resources) (*Machine, error) {
	if resources.Memory == nil || resources.Time == nil {
		return nil, errors.New("memory and time counter are required")
	}
	if len(args) != len(fn.Params) {
		return nil, fmt.Errorf("function requires %d arguments, got %d", len(fn.Params), len(args))
	}
	for i, param := range fn.Params {
		if param.Ident.Variable != bot.VariableIndex(i) || !matches(resources.Memory, args[i], param.Type) {
			return nil, fmt.Errorf("invalid argument or parameter slot %d", i)
		}
	}
	m := &Machine{resources: resources, owned: make(map[memory.ID]uint64), source: fn.Source}
	for i, id := range args {
		if err := resources.Memory.Retain(id); err != nil {
			return nil, errors.Join(err, m.Close())
		}
		m.owned[id]++
		m.variables = append(m.variables, binding{value: id, hint: fn.Params[i].Ident.HintID})
	}
	m.push(enterBlock{block: fn.Body})
	for i := len(fn.Params) - 1; i >= 0; i-- {
		m.push(parameter{index: i})
	}
	return m, nil
}

// Step executes one continuation, allowing the owner to enforce limits even
// inside loops that never call another function. Runtime failures release the
// invocation's references and leave emitted events available in the collector.
func (m *Machine) Step() (state State, err error) {
	if m == nil || m.owned == nil || m.phase != executing {
		return nil, errors.New("function machine is not ready to step")
	}
	defer func() {
		if failure := recover(); failure != nil {
			cause, ok := failure.(error)
			if !ok {
				cause = fmt.Errorf("%v", failure)
			}
			err = &RuntimeError{Source: m.source, Err: errors.Join(cause, m.Close())}
			state = nil
		}
	}()
	if len(m.tasks) == 0 {
		panic(errors.New("function ended without returning a value"))
	}
	next := m.tasks[len(m.tasks)-1]
	m.tasks[len(m.tasks)-1] = nil
	m.tasks = m.tasks[:len(m.tasks)-1]
	next.run(m)
	switch m.phase {
	case awaiting:
		call := m.waiting
		call.Args = slices.Clone(call.Args)
		return call, nil
	case finished:
		return Returned{Value: m.result}, nil
	default:
		return Running{}, nil
	}
}

// Resume retains a call's result before releasing the borrowed arguments, so a
// result may share an argument's ID. The caller keeps its reference to result.
// Rejected resumes leave the suspended invocation unchanged.
func (m *Machine) Resume(result memory.ID) error {
	if m == nil || m.phase != awaiting {
		return errors.New("function machine is not awaiting a call result")
	}
	if !matches(m.resources.Memory, result, m.waiting.ResultType) {
		return errors.New("call result does not match its checked type")
	}
	if err := m.resources.Memory.Retain(result); err != nil {
		return err
	}
	m.owned[result]++
	for _, id := range m.waiting.Args {
		m.drop(id)
	}
	m.current(m.waiting.Source)
	m.waiting = Call{}
	m.operands = append(m.operands, result)
	m.phase = executing
	return nil
}

// Close is idempotent. A reference already transferred by Returned belongs to
// the caller and is not released here.
func (m *Machine) Close() error {
	if m == nil || m.phase == closed {
		return nil
	}
	var errs []error
	for i := len(m.variables) - 1; i >= 0; i-- {
		if m.variables[i].announced {
			m.resources.Events.Add(robotevent.VariableDropEvent{HintID: m.variables[i].hint})
		}
	}
	for id, count := range m.owned {
		for range count {
			if err := m.resources.Memory.Release(id); err != nil {
				errs = append(errs, err)
			}
		}
	}
	m.tasks, m.operands, m.variables = nil, nil, nil
	clear(m.owned)
	m.waiting = Call{}
	m.phase = closed
	return errors.Join(errs...)
}

func (m *Machine) push(t task) { m.tasks = append(m.tasks, t) }

func (m *Machine) current(source bot.Source) {
	m.source = source
	if m.resources.Events != nil {
		m.resources.Events.Add(robotevent.CurrentNodeEvent{HintID: source.HintID})
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func (m *Machine) allocate(value memory.Value) memory.ID {
	id, err := m.resources.Memory.Add(value)
	must(err)
	m.owned[id]++
	return id
}

func (m *Machine) retain(id memory.ID) memory.ID {
	must(m.resources.Memory.Retain(id))
	m.owned[id]++
	return id
}

func (m *Machine) forget(id memory.ID) {
	if m.owned[id] == 0 {
		panic(fmt.Errorf("machine does not own value %d", id))
	}
	m.owned[id]--
	if m.owned[id] == 0 {
		delete(m.owned, id)
	}
}

func (m *Machine) drop(id memory.ID) {
	must(m.resources.Memory.Release(id))
	m.forget(id)
}

func (m *Machine) pop() memory.ID {
	id := m.operands[len(m.operands)-1]
	m.operands = m.operands[:len(m.operands)-1]
	return id
}

func (m *Machine) value(id memory.ID) memory.Value {
	value, exists := m.resources.Memory.Get(id)
	if !exists {
		panic(fmt.Errorf("unknown memory value %d", id))
	}
	return value
}

func (m *Machine) leave(base int) {
	for i := len(m.variables) - 1; i >= base; i-- {
		variable := m.variables[i]
		if variable.announced {
			m.resources.Events.Add(robotevent.VariableDropEvent{HintID: variable.hint})
		}
		m.drop(variable.value)
	}
	m.variables = m.variables[:base]
}
