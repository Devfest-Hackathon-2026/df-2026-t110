// Package board exposes the grid-based game board.
package board

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/board/internal/engine"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
)

var ErrTickLimit = engine.ErrTickLimit

type (
	Config     = engine.Config
	Snapshot   = engine.Snapshot
	Position   = engine.Position
	Direction  = engine.Direction
	RobotState = engine.RobotState
	TeamID     = engine.TeamID

	Terrain        = engine.Terrain
	LandTerrain    = engine.LandTerrain
	RockTerrain    = engine.RockTerrain
	SeaTerrain     = engine.SeaTerrain
	LandKind       = engine.LandKind
	PlainLand      = engine.PlainLand
	GoalLand       = engine.GoalLand
	CheckpointLand = engine.CheckpointLand
)

const (
	North = engine.North
	East  = engine.East
	South = engine.South
	West  = engine.West

	TeamID1 = engine.TeamID1
	TeamID2 = engine.TeamID2
)

type Board struct {
	impl engine.Board
}

func New(config Config) (*Board, error) {
	impl, err := engine.New(config)
	if err != nil {
		return nil, err
	}
	return &Board{impl: impl}, nil
}

func (b *Board) Snapshot() Snapshot {
	return b.impl.Snapshot()
}

func (b *Board) Start(ids []protocol.RobotID) ([]fieldevent.FieldEvent, error) {
	return b.impl.Start(ids)
}

func (b *Board) Next(reply robotmsg.Reply) (boardmsg.BoardOutput, error) {
	return b.impl.Next(reply)
}

func NewPosition(x, y int) Position {
	return engine.NewPosition(x, y)
}

func NewRobotState(position Position, direction Direction, drillDurability int64) RobotState {
	return engine.NewRobotState(position, direction, drillDurability)
}

func NewLandTerrain(kind LandKind, hasLuggage bool, territoryOwner TeamID) LandTerrain {
	return engine.NewLandTerrain(kind, hasLuggage, territoryOwner)
}

func NewCheckpointLand(order int64) CheckpointLand {
	return engine.NewCheckpointLand(order)
}
