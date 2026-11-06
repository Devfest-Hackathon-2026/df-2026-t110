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

// Recorder writes the inputs and outputs exchanged between a hub and its board
// and robots. One recorder represents one run.
//
// The caller records every completed Start or Next call, including failed calls,
// synchronously in call order. Outputs are recorded as returned, before protocol
// validation. A nil reply to Board.Next is also an input to record.
//
// Implementations must not modify the supplied values and must copy any mutable
// data they retain after returning. The caller owns the recorder's lifetime and
// is responsible for flushing or closing its underlying storage.
type Recorder interface {
	// RecordBoardStart records Board.Start's robot IDs and returned error.
	// callErr belongs to the board call; the returned error is a logging failure.
	RecordBoardStart(ctx context.Context, ids []RobotID, callErr error) error

	// RecordBoardNext records Board.Next's reply, decision, and returned error.
	// callErr belongs to the board call; the returned error is a logging failure.
	RecordBoardNext(ctx context.Context, reply Reply, decision Decision, callErr error) error

	// RecordRobotStart records the identified robot's Start result and error.
	// callErr belongs to the robot call; the returned error is a logging failure.
	RecordRobotStart(ctx context.Context, id RobotID, step Step, callErr error) error

	// RecordRobotNext records the identified robot's Next input, result, and error.
	// callErr belongs to the robot call; the returned error is a logging failure.
	RecordRobotNext(ctx context.Context, id RobotID, value Value, step Step, callErr error) error
}
