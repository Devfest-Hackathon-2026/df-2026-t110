// Package game defines the contracts shared by robots, boards, and the hub.
package game

import (
	"context"
	"errors"
)

// ErrNotSupported identifies actions a concrete implementation does not support.
var ErrNotSupported = errors.New("not supported")

type RobotID string

type ValueKind string

const (
	Bool    ValueKind = "bool"
	Integer ValueKind = "integer"
	Unit    ValueKind = "unit"
	Field   ValueKind = "field"
)

type Terrain string

const (
	Land Terrain = "land"
	Rock Terrain = "rock"
	Sea  Terrain = "sea"
)

// FieldValue stores one cell's terrain and object attributes at observation
// time. It is copied by value so later board changes do not affect it.
type FieldValue struct {
	Terrain    Terrain
	Goal       bool
	Checkpoint bool
	Luggage    bool
}

// Value is a builtin's return value, never a board snapshot or a tick. Kind
// selects Bool, Integer, Field, or no payload for Unit.
type Value struct {
	Kind    ValueKind
	Bool    bool
	Integer int64
	Field   FieldValue
}

type ActionKind string

const (
	Forward         ActionKind = "forward"
	Back            ActionKind = "back"
	Rotate          ActionKind = "rotate"
	Wait            ActionKind = "wait"
	PickUp          ActionKind = "pick_up"
	Release         ActionKind = "release"
	Drill           ActionKind = "drill"
	DrillDurability ActionKind = "drill_durability"
	Distance        ActionKind = "distance"
	SearchFront     ActionKind = "search_front"
	SearchNow       ActionKind = "search_now"
	IsLand          ActionKind = "is_land"
	IsGoal          ActionKind = "is_goal"
	IsCheckpoint    ActionKind = "is_checkpoint"
	IsLaggage       ActionKind = "is_laggage"
	IsSea           ActionKind = "is_sea"
	IsRock          ActionKind = "is_rock"
	PrintInteger    ActionKind = "print_integer"
	PrintBool       ActionKind = "print_bool"
)

// Rotation is a relative quarter-turn, matching picolang's rotation type.
type Rotation string

const (
	Left  Rotation = "left"
	Right Rotation = "right"
)

// Action covers the action builtins in Docs/picolang.md. Kind selects the
// argument: Rotation for Rotate, Integer for Wait/PrintInteger, Bool for PrintBool,
// Field for is_* actions, and no payload for the remaining actions. Unused
// argument fields must be zero. Tick costs and execution belong to the board.
type Action struct {
	Kind     ActionKind
	Rotation Rotation
	Integer  int64
	Bool     bool
	Field    FieldValue
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
