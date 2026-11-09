// Package field_event defines the board data recorded for replay.
package field_event

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

type FieldEvent struct {
	Tick  uint64
	State FieldState
}

type FieldState interface{ fieldState() }

// InitialState contains the full board at tick zero. Terrain is indexed by [y][x].
type InitialState struct {
	Terrain [][]Cell
	Robots  []RobotState

	FieldState
}

// UpdateState records the results of one committed turn or removals after the last turn.
type UpdateState struct {
	Actions         []ActionResult
	RemovedRobotIDs []protocol.RobotID

	FieldState
}

// Cell records one terrain cell and its land attributes.
// The engine's Terrain types keep these values private and enforce board invariants.
type Cell struct {
	Type           string
	Kind           *LandKind
	HasLuggage     bool
	TerritoryOwner int
}

type LandKind struct {
	Type  string
	Order *int64
}

type Position struct {
	X int
	Y int
}

type RobotState struct {
	ID              protocol.RobotID
	Position        Position
	Direction       string
	DrillDurability int64
	TeamID          int
	CarryingLuggage bool
}

type ActionResult interface{ actionResult() }

type MoveKind string

const (
	Forward MoveKind = "forward"
	Back    MoveKind = "back"
)

type MoveResult struct {
	Kind    MoveKind
	RobotID protocol.RobotID
	Success bool
	To      *Position

	ActionResult
}

type RotateResult struct {
	RobotID   protocol.RobotID
	Rotation  values.Rotation
	Direction string

	ActionResult
}

type PrintIntegerResult struct {
	RobotID protocol.RobotID
	Integer int64

	ActionResult
}

type PrintBoolResult struct {
	RobotID protocol.RobotID
	Bool    bool

	ActionResult
}

type DrillResult struct {
	RobotID         protocol.RobotID
	Success         bool
	Target          *Position
	DrillDurability int64

	ActionResult
}

type PickUpResult struct {
	RobotID protocol.RobotID
	Success bool
	Target  *Position

	ActionResult
}

type ReleaseResult struct {
	RobotID   protocol.RobotID
	Success   bool
	Target    *Position
	Discarded bool

	ActionResult
}

type WaitResult struct {
	RobotID protocol.RobotID
	Ticks   int64

	ActionResult
}
