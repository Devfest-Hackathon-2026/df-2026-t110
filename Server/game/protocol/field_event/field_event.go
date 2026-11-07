// Package field_event defines the board data recorded for replay.
package field_event

import (
	json "encoding/json/v2"
	"fmt"

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
	Terrain [][]Cell     `json:"terrain"`
	Robots  []RobotState `json:"robots"`

	FieldState `json:"-"`
}

// UpdateState records the results of one committed turn or removals after the last turn.
type UpdateState struct {
	Actions         []ActionResult     `json:"actions,omitempty"`
	RemovedRobotIDs []protocol.RobotID `json:"removedRobotIds,omitempty"`

	FieldState `json:"-"`
}

// Cell is the JSON representation of one terrain cell and its land attributes.
// The engine's Terrain types keep these values private and enforce board invariants.
type Cell struct {
	Type           string    `json:"type"`
	Kind           *LandKind `json:"kind,omitempty"`
	HasLuggage     bool      `json:"hasLuggage,omitzero"`
	TerritoryOwner int       `json:"territoryOwner,omitzero"`
}

type LandKind struct {
	Type  string `json:"type"`
	Order *int64 `json:"order,omitempty"`
}

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type RobotState struct {
	ID              protocol.RobotID `json:"id"`
	Position        Position         `json:"position"`
	Direction       string           `json:"direction"`
	DrillDurability int64            `json:"drillDurability"`
}

type ActionResult interface{ actionResult() }

type MoveKind string

const (
	Forward MoveKind = "forward"
	Back    MoveKind = "back"
)

type MoveResult struct {
	Kind    MoveKind         `json:"kind"`
	RobotID protocol.RobotID `json:"robotId"`
	Success bool             `json:"success"`
	To      *Position        `json:"to,omitempty"`

	ActionResult `json:"-"`
}

type RotateResult struct {
	RobotID   protocol.RobotID `json:"robotId"`
	Rotation  values.Rotation  `json:"rotation"`
	Direction string           `json:"direction"`

	ActionResult `json:"-"`
}

func (r RotateResult) MarshalJSON() ([]byte, error) {
	type payload RotateResult
	return json.Marshal(struct {
		Kind string `json:"kind"`
		payload
	}{Kind: "rotate", payload: payload(r)})
}

func (e FieldEvent) MarshalJSON() ([]byte, error) {
	var payload any
	switch state := e.State.(type) {
	case InitialState:
		payload = struct {
			Kind string `json:"kind"`
			InitialState
		}{Kind: "initial", InitialState: state}
	case UpdateState:
		payload = struct {
			Kind string `json:"kind"`
			UpdateState
		}{Kind: "update", UpdateState: state}
	default:
		return nil, fmt.Errorf("invalid field state %T", e.State)
	}
	return json.Marshal(struct {
		Type  string `json:"type"`
		Tick  uint64 `json:"tick"`
		State any    `json:"state"`
	}{Type: "field", Tick: e.Tick, State: payload})
}
