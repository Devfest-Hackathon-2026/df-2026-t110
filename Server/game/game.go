// Package game defines the contracts shared by robots, boards, and the hub.
package game

import "context"

type RobotID string

type ValueKind string

const (
	Bool    ValueKind = "bool"
	Integer ValueKind = "integer"
	Unit    ValueKind = "unit"
)

// Value is a builtin's return value, never a board snapshot or a tick.
type Value struct {
	Kind    ValueKind
	Bool    bool
	Integer int64
}

type ActionKind string

const (
	Forward         ActionKind = "forward"
	Back            ActionKind = "back"
	Rotate          ActionKind = "rotate"
	DrillDurability ActionKind = "drill_durability"
)

type Turn string

const (
	Left  Turn = "left"
	Right Turn = "right"
)

type Action struct {
	Kind ActionKind
	Turn Turn
}

// Step must contain exactly one of Action or Done.
type Step struct {
	Action *Action
	Done   bool
}

type Robot interface {
	Next(context.Context, *Value) (Step, error)
}

// Reply must contain exactly one of Action, Done, or Err.
type Reply struct {
	RobotID RobotID
	Action  *Action
	Done    bool
	Err     error
}

type Request struct {
	RobotID     RobotID
	ReturnValue *Value
}

type ActionResult struct {
	RobotID RobotID
	Action  Action
	Value   Value
}

type TickUpdate struct {
	Tick    uint64
	Results []ActionResult
}

// Decision contains exactly one event. AdvanceTo proposes a future tick;
// calling Next(ctx, nil) authorizes its atomic commit. This lets the hub check
// its tick budget and cancellation before any pending changes are applied.
type Decision struct {
	Request   *Request
	AdvanceTo *uint64
	Update    *TickUpdate
	Done      bool
}

// Board selects robots and owns the clock. After a Request, Next requires that
// robot's Reply. After AdvanceTo or Update, Next requires nil. The first Next
// after Start also requires nil. Instances are used for one run only.
type Board interface {
	Start(context.Context, []RobotID) error
	Next(context.Context, *Reply) (Decision, error)
}
