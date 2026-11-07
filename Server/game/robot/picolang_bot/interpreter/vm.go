// Package interpreter runs checked picolang programs using an explicit call stack.
package interpreter

import (
	"errors"
	"fmt"
	"maps"

	"github.com/dvpico/Programming-Learning-AR/Server/game"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/builtin"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/event"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/function"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
	clock "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/time"
)

var ErrState = errors.New("invalid interpreter state")

type phase uint8

const (
	fresh phase = iota
	running
	awaiting
	finished
	failed
	closed
)

// VM owns one program run. New constructs a fresh VM; Start/Next run until an
// action, completion, or error and return the collected events. Execution and
// resource bookkeeping are internal to the VM. Calls must be sequential; do not
// copy a VM or mutate its checked AST. Close abandons a suspended run.
type VM struct {
	program     bot.TypedProgram
	limits      Limits
	source      bot.Source
	memory      memory.Store
	time        clock.Counter
	events      event.Collector
	evaluator   *builtin.Evaluator
	stack       []*function.Machine
	pending     builtin.Action
	phase       phase
	initialized bool
}

var _ game.Robot = (*VM)(nil)

// New accepts only a successfully checked program. The function table is copied;
// the AST it contains remains borrowed and must not be modified. The seed creates
// one random stream shared across every call and action suspension in this run.
// Resource limits must be positive and apply across the entire run.
func New(program bot.TypedProgram, seed uint64, limits Limits) (*VM, error) {
	if limits.MaxSteps == 0 || limits.MaxMemoryBytes == 0 || limits.MaxCallDepth == 0 {
		return nil, errors.New("VM resource limits must be positive")
	}
	main, exists := program.Functions["main"]
	_, unit := bot.NodeValue(main.ReturnType).(bot.UnitType)
	if !exists || len(main.Params) != 0 || !unit {
		return nil, errors.New("main must have signature () -> unit")
	}
	v := &VM{program: bot.TypedProgram{Functions: maps.Clone(program.Functions)}, limits: limits, source: main.Source, initialized: true}
	evaluator, err := builtin.New(&v.memory, seed)
	if err != nil {
		return nil, err
	}
	v.evaluator = evaluator
	return v, nil
}

func (v *VM) resources() function.Resources {
	return function.Resources{Memory: &v.memory, Time: &v.time, Events: &v.events}
}

// step executes one transition, including pushing or popping at most one
// invocation. Its first call starts main. It never executes a callee recursively
// on the Go stack. A runtime failure closes all active frames, innermost first.
func (v *VM) step() (robotmsg.Step, error) {
	if v.phase == fresh {
		main, err := function.New(v.program.Functions["main"], nil, v.resources())
		if err != nil {
			return nil, v.fail(err)
		}
		v.time.Increment() // The entry-point invocation has no caller to charge it.
		v.stack = append(v.stack, main)
		v.phase = running
		return nil, nil
	}
	frame := v.stack[len(v.stack)-1]
	state, err := frame.Step()
	v.source = frame.Source()
	if err != nil {
		return nil, v.fail(err)
	}
	// Check charged work before evaluating a call or returning an action.
	if err := v.checkLimits(); err != nil {
		return nil, v.fail(err)
	}
	switch state := state.(type) {
	case function.Running:
		return nil, nil
	case function.Call:
		if fn, exists := v.program.Functions[state.Func]; exists {
			callee, err := function.New(fn, state.Args, v.resources())
			if err != nil {
				return nil, v.fail(&function.RuntimeError{Source: state.Source, Err: err})
			}
			v.stack = append(v.stack, callee)
			return nil, nil
		}
		result, err := v.evaluator.Evaluate(state.Func, state.Args)
		if err != nil {
			return nil, v.fail(&function.RuntimeError{Source: state.Source, Err: err})
		}
		switch result := result.(type) {
		case builtin.Immediate:
			if err := v.acceptResult(result.Value); err != nil {
				return nil, v.fail(err)
			}
			return nil, nil
		case builtin.Action:
			v.pending, v.phase = result, awaiting
			return robotmsg.ActionStep{Action: result.Action}, nil
		}
	case function.Returned:
		v.stack[len(v.stack)-1] = nil
		v.stack = v.stack[:len(v.stack)-1]
		if err := frame.Close(); err != nil {
			return nil, v.fail(errors.Join(err, v.memory.Release(state.Value)))
		}
		if len(v.stack) == 0 {
			value, _ := v.memory.Get(state.Value)
			_, unit := value.(memory.Unit)
			err := v.memory.Release(state.Value)
			if !unit {
				err = errors.Join(errors.New("main returned a non-unit value"), err)
			}
			if err != nil {
				return nil, v.fail(err)
			}
			v.phase = finished
			return robotmsg.DoneStep{}, nil
		}
		if err := v.acceptResult(state.Value); err != nil {
			return nil, v.fail(err)
		}
		return nil, nil
	}
	return nil, v.fail(fmt.Errorf("unexpected function state %T", state))
}

// acceptResult consumes the evaluator's or callee's owned return reference.
// Resume retains it before the temporary reference is released, including when
// caller arguments and the returned value share an ID.
func (v *VM) acceptResult(id memory.ID) error {
	err := v.stack[len(v.stack)-1].Resume(id)
	return errors.Join(err, v.memory.Release(id))
}

// resume supplies a board result to the action currently suspending execution.
// Incorrect results leave the action and stack intact so the owner may retry.
func (v *VM) resume(result values.Value) error {
	if v == nil || !v.initialized || v.phase != awaiting {
		return fmt.Errorf("%w: VM is not awaiting an action result", ErrState)
	}
	id, err := v.evaluator.Complete(v.pending, result)
	if err != nil {
		return err
	}
	// Completion can allocate a board observation. Check before Resume releases
	// the caller's arguments, and release the temporary reference on failure.
	if err := v.checkLimits(); err != nil {
		return v.fail(errors.Join(err, v.memory.Release(id)))
	}
	if err := v.acceptResult(id); err != nil {
		return v.fail(err)
	}
	v.pending, v.phase = builtin.Action{}, running
	return nil
}

// Collect transfers pending events. Start and Next call it automatically before
// returning. It can also collect cleanup events after Close.
func (v *VM) Collect() []robotevent.PicolangEvent {
	if v == nil {
		return nil
	}
	return v.events.Collect()
}

func (v *VM) closeFrames() error {
	var errs []error
	for i := len(v.stack) - 1; i >= 0; i-- {
		if err := v.stack[i].Close(); err != nil {
			errs = append(errs, err)
		}
	}
	v.stack = nil
	v.pending = builtin.Action{}
	return errors.Join(errs...)
}

func (v *VM) fail(err error) error {
	err = errors.Join(err, v.closeFrames())
	v.phase = failed
	return err
}

// Close is idempotent and retains cleanup events for Collect. It also closes
// caller frames suspended under the active invocation.
func (v *VM) Close() error {
	if v == nil || v.phase == closed {
		return nil
	}
	err := v.closeFrames()
	v.phase = closed
	return err
}

func (v *VM) Start() (robotmsg.RobotOutput, error) {
	if v == nil || !v.initialized || v.phase != fresh {
		return robotmsg.RobotOutput{}, fmt.Errorf("%w: VM must be fresh to start", ErrState)
	}
	return v.run()
}

func (v *VM) Next(result values.Value) (robotmsg.RobotOutput, error) {
	if err := v.resume(result); err != nil {
		return robotmsg.RobotOutput{Events: v.Collect()}, err
	}
	return v.run()
}

func (v *VM) run() (robotmsg.RobotOutput, error) {
	if err := v.checkLimits(); err != nil {
		err = v.fail(err)
		return robotmsg.RobotOutput{Events: v.Collect()}, err
	}
	for {
		step, err := v.step()
		if err == nil {
			if limitErr := v.checkLimits(); limitErr != nil {
				err = v.fail(limitErr)
			}
		}
		if err != nil {
			return robotmsg.RobotOutput{Events: v.Collect()}, err
		}
		if step != nil {
			return robotmsg.RobotOutput{Step: step, Events: v.Collect()}, nil
		}
	}
}
