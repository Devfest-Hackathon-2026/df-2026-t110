// Package function executes one checked picolang function as a resumable machine.
package function

import (
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/event"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
	clock "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/time"
)

// Resources are shared by the machines in one interpreter. Calls must be
// sequential. The owner checks execution limits between Step calls.
type Resources struct {
	Memory *memory.Store
	Time   *clock.Counter
	Events *event.Collector // nil disables execution events.
}

type State interface{ functionState() }

// Running means another local transition can be executed with Step.
type Running struct{}

// Call requests execution outside this machine, for both user functions and
// builtins. Args are borrowed until Resume or Close; retain any needed longer.
// The argument slice belongs to the caller. ResultType comes from type checking.
type Call struct {
	bot.Source
	Func       bot.Name
	Args       []memory.ID
	ResultType bot.Type
}

// Returned transfers one owned reference to the caller, who must release it.
type Returned struct{ Value memory.ID }

func (Running) functionState()  {}
func (Call) functionState()     {}
func (Returned) functionState() {}

// RuntimeError reports a failed transition and the source being executed.
type RuntimeError struct {
	bot.Source
	Err error
}

func (e *RuntimeError) Error() string {
	return "function execution at hint " + string(e.HintID) + ": " + e.Err.Error()
}

func (e *RuntimeError) Unwrap() error { return e.Err }
