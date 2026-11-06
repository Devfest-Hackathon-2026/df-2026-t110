// Package game defines the contracts shared by robots, boards, and the hub.
package game

import (
	"context"
	"errors"
)

// ErrNotSupported identifies actions a concrete implementation does not support.
var ErrNotSupported = errors.New("not supported")

type RobotID string

// Rotation is a relative quarter-turn, matching picolang's rotation type.
type Rotation string

const (
	Left  Rotation = "left"
	Right Rotation = "right"
)

type Step interface{ step() }

type ActionStep struct{ Action Action }
type DoneStep struct{}

func (ActionStep) step() {}
func (DoneStep) step()   {}

// Start runs a robot without an input value. Next resumes it with the return
// value of its previous action.
type Robot interface {
	Start(context.Context) (Step, error)
	Next(context.Context, Value) (Step, error)
}

type Reply interface{ reply() }

type ActionReply struct {
	RobotID RobotID
	Action  Action
}

type DoneReply struct{ RobotID RobotID }

type ErrorReply struct {
	RobotID RobotID
	Err     error
}

func (ActionReply) reply() {}
func (DoneReply) reply()   {}
func (ErrorReply) reply()  {}

type ActionResult struct {
	RobotID RobotID
	Action  Action
	Value   Value
}

type TickUpdate struct {
	Tick    uint64
	Results []ActionResult
}

// Decision describes one board event. AdvanceDecision proposes a future tick;
// calling Next(ctx, nil) authorizes its atomic commit. This lets the hub check
// its tick budget and cancellation before any pending changes are applied.
type Decision interface{ decision() }

type StartDecision struct{ RobotID RobotID }

type ResumeDecision struct {
	RobotID     RobotID
	ReturnValue Value
}

type AdvanceDecision struct{ Tick uint64 }
type UpdateDecision struct{ Update TickUpdate }
type DoneDecision struct{}

func (StartDecision) decision()   {}
func (ResumeDecision) decision()  {}
func (AdvanceDecision) decision() {}
func (UpdateDecision) decision()  {}
func (DoneDecision) decision()    {}

// Board selects robots and owns the clock. After StartDecision or
// ResumeDecision, Next requires that robot's Reply. After AdvanceDecision or
// UpdateDecision, Next requires nil. The first Next after Start also requires
// nil. Instances are used for one run only.
type Board interface {
	Start(context.Context, []RobotID) error
	Next(context.Context, Reply) (Decision, error)
}
