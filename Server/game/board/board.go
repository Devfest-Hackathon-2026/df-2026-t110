// Package board provides a deterministic grid board and logical-tick scheduler.
package board

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/dvpico/Programming-Learning-AR/Server/game"
)

type Direction string

const (
	North Direction = "north"
	East  Direction = "east"
	South Direction = "south"
	West  Direction = "west"
)

type Position struct{ X, Y int }

var directionDeltas = map[Direction]Position{
	North: {0, -1},
	East:  {1, 0},
	South: {0, 1},
	West:  {-1, 0},
}

var directionIndices = map[Direction]int{
	North: 0,
	East:  1,
	South: 2,
	West:  3,
}

type RobotState struct {
	Position        Position
	Direction       Direction
	DrillDurability int64
}

type Config struct {
	Terrain [][]game.TerrainValue
	Robots  map[game.RobotID]RobotState
}

type Snapshot struct {
	Tick    uint64
	Terrain [][]game.TerrainValue
	Robots  map[game.RobotID]RobotState
}

type Board struct {
	state    Snapshot
	order    []game.RobotID
	controls map[game.RobotID]control
	phase    phase
}

func New(config Config) (*Board, error) {
	if len(config.Terrain) == 0 || len(config.Terrain[0]) == 0 || len(config.Robots) == 0 {
		return nil, errors.New("nonempty terrain and robots are required")
	}
	width := len(config.Terrain[0])
	for _, row := range config.Terrain {
		if len(row) != width {
			return nil, errors.New("terrain must be rectangular")
		}
		for _, cell := range row {
			if !game.ValidValue(cell) {
				return nil, fmt.Errorf("invalid terrain %T", cell)
			}
		}
	}
	occupied := make(map[Position]struct{}, len(config.Robots))
	for id, robot := range config.Robots {
		p := robot.Position
		if id == "" || directionIndex(robot.Direction) < 0 || robot.DrillDurability < 0 ||
			p.X < 0 || p.Y < 0 || p.X >= width || p.Y >= len(config.Terrain) {
			return nil, fmt.Errorf("invalid robot %q", id)
		}
		_, isLand := config.Terrain[p.Y][p.X].(game.LandTerrain)
		if _, exists := occupied[p]; !isLand || exists {
			return nil, fmt.Errorf("robot %q must occupy a unique land cell", id)
		}
		occupied[p] = struct{}{}
	}
	state := Snapshot{Terrain: config.Terrain, Robots: config.Robots}
	return &Board{state: clone(state), phase: unstartedBoard{}}, nil
}

func (b *Board) Snapshot() Snapshot { return clone(b.state) }

func (b *Board) Start(ctx context.Context, ids []game.RobotID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, fresh := b.phase.(unstartedBoard); !fresh || len(ids) != len(b.state.Robots) {
		return errors.New("board is single-use and robot IDs must match its configuration")
	}
	controls := make(map[game.RobotID]control, len(ids))
	for _, id := range ids {
		if _, ok := b.state.Robots[id]; !ok || controls[id] != nil {
			return fmt.Errorf("unknown or duplicate robot %q", id)
		}
		controls[id] = unstartedControl{}
	}
	b.controls, b.order, b.phase = controls, append([]game.RobotID(nil), ids...), collectingBoard{}
	return nil
}

func (b *Board) Next(ctx context.Context, reply game.Reply) (game.Decision, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch phase := b.phase.(type) {
	case unstartedBoard:
		return nil, errors.New("board has not started")
	case awaitingReplyBoard:
		if err := b.accept(phase.robotID, reply); err != nil {
			return nil, err
		}
		b.phase = collectingBoard{}
	case collectingBoard, awaitingCommitBoard, finishedBoard:
		if reply != nil {
			return nil, errors.New("unexpected robot reply")
		}
	default:
		return nil, errors.New("invalid board phase")
	}
	switch b.phase.(type) {
	case awaitingCommitBoard:
		update, err := b.commit(ctx)
		if err != nil {
			return nil, err
		}
		b.phase = collectingBoard{}
		return game.UpdateDecision{Update: update}, nil
	case finishedBoard:
		return game.DoneDecision{}, nil
	}
	pending := false
	for _, id := range b.order {
		switch c := b.controls[id].(type) {
		case pendingControl:
			pending = true
		case unstartedControl:
			b.phase = awaitingReplyBoard{robotID: id}
			return game.StartDecision{RobotID: id}, nil
		case readyControl:
			b.phase = awaitingReplyBoard{robotID: id}
			return game.ResumeDecision{RobotID: id, ReturnValue: c.value}, nil
		case finishedControl:
		default:
			return nil, fmt.Errorf("invalid control for robot %q", id)
		}
	}
	if pending {
		if b.state.Tick == math.MaxUint64 {
			return nil, errors.New("tick overflow")
		}
		tick := b.state.Tick + 1
		b.phase = awaitingCommitBoard{}
		return game.AdvanceDecision{Tick: tick}, nil
	}
	b.phase = finishedBoard{}
	return game.DoneDecision{}, nil
}

func (b *Board) accept(id game.RobotID, reply game.Reply) error {
	var robotID game.RobotID
	switch r := reply.(type) {
	case game.ActionReply:
		robotID = r.RobotID
	case game.DoneReply:
		robotID = r.RobotID
	case game.ErrorReply:
		robotID = r.RobotID
	default:
		return errors.New("reply must contain an action, completion, or error")
	}
	if robotID != id {
		return errors.New("reply must match the requested robot")
	}
	switch r := reply.(type) {
	case game.ActionReply:
		if r.Action == nil {
			return errors.New("action reply must contain an action")
		}
		switch a := r.Action.(type) {
		case game.DrillDurabilityAction:
			b.controls[id] = readyControl{value: game.IntegerValue{Integer: b.state.Robots[id].DrillDurability}}
		case game.ForwardAction, game.BackAction:
			b.controls[id] = pendingControl{action: a}
		case game.RotateAction:
			if a.Rotation != game.Left && a.Rotation != game.Right {
				return errors.New("rotate requires left or right")
			}
			b.controls[id] = pendingControl{action: a}
		default:
			return fmt.Errorf("%w: action %T", game.ErrNotSupported, a)
		}
	case game.DoneReply:
		b.controls[id] = finishedControl{}
	case game.ErrorReply:
		if r.Err == nil {
			return errors.New("error reply must contain an error")
		}
		b.controls[id] = finishedControl{}
	}
	return nil
}

// commit resolves movement dependencies against one immutable starting state.
func (b *Board) commit(ctx context.Context) (game.TickUpdate, error) {
	robotCount := len(b.state.Robots)
	occupied := make(map[Position]game.RobotID, robotCount)
	targets := make(map[game.RobotID]Position, robotCount)
	counts := make(map[Position]int, robotCount)
	for id, robot := range b.state.Robots {
		occupied[robot.Position] = id
	}
	for _, id := range b.order {
		c, pending := b.controls[id].(pendingControl)
		if !pending {
			continue
		}
		a := c.action
		robot := b.state.Robots[id]
		delta := directionDeltas[robot.Direction]
		switch a.(type) {
		case game.ForwardAction:
		case game.BackAction:
			delta.X, delta.Y = -delta.X, -delta.Y
		default:
			continue
		}
		target := Position{robot.Position.X + delta.X, robot.Position.Y + delta.Y}
		if b.land(target) {
			targets[id] = target
			counts[target]++
		}
	}
	visiting := make(map[game.RobotID]struct{}, robotCount)
	resolved := make(map[game.RobotID]struct{}, robotCount)
	movable := make(map[game.RobotID]bool, robotCount)
	var canMove func(game.RobotID) bool
	canMove = func(id game.RobotID) bool {
		if _, exists := resolved[id]; exists {
			return movable[id]
		}
		if _, exists := visiting[id]; exists {
			// Swaps are rejected below, so this is a longer cycle.
			return true
		}
		target, ok := targets[id]
		if !ok || counts[target] != 1 {
			resolved[id] = struct{}{}
			return false
		}
		other, occupied := occupied[target]
		if reverse, moving := targets[other]; occupied && moving && reverse == b.state.Robots[id].Position {
			resolved[id] = struct{}{}
			return false
		}
		visiting[id] = struct{}{}
		movable[id] = !occupied || canMove(other)
		delete(visiting, id)
		resolved[id] = struct{}{}
		return movable[id]
	}
	next := clone(b.state)
	next.Tick++
	update := game.TickUpdate{Tick: next.Tick}
	for _, id := range b.order {
		c, pending := b.controls[id].(pendingControl)
		if !pending {
			continue
		}
		a := c.action
		robot := next.Robots[id]
		var value game.Value = game.UnitValue{}
		switch action := a.(type) {
		case game.RotateAction:
			rotation := 1
			if action.Rotation == game.Left {
				rotation = 3
			}
			robot.Direction = [...]Direction{North, East, South, West}[(directionIndex(robot.Direction)+rotation)%4]
		case game.ForwardAction, game.BackAction:
			moved := canMove(id)
			value = game.BoolValue{Bool: moved}
			if moved {
				robot.Position = targets[id]
			}
		default:
			return game.TickUpdate{}, fmt.Errorf("%w: pending action %T", game.ErrNotSupported, a)
		}
		next.Robots[id] = robot
		update.Results = append(update.Results, game.ActionResult{RobotID: id, Action: a, Value: value})
	}
	if err := ctx.Err(); err != nil {
		return game.TickUpdate{}, err
	}
	b.state = next
	for _, result := range update.Results {
		b.controls[result.RobotID] = readyControl{value: result.Value}
	}
	return update, nil
}

func (b *Board) land(p Position) bool {
	if p.Y < 0 || p.Y >= len(b.state.Terrain) || p.X < 0 || p.X >= len(b.state.Terrain[0]) {
		return false
	}
	_, isLand := b.state.Terrain[p.Y][p.X].(game.LandTerrain)
	return isLand
}

func directionIndex(d Direction) int {
	if index, ok := directionIndices[d]; ok {
		return index
	}
	return -1
}

func clone(state Snapshot) Snapshot {
	copy := Snapshot{Tick: state.Tick, Terrain: make([][]game.TerrainValue, len(state.Terrain)), Robots: make(map[game.RobotID]RobotState, len(state.Robots))}
	for y, row := range state.Terrain {
		copy.Terrain[y] = append([]game.TerrainValue(nil), row...)
	}
	for id, robot := range state.Robots {
		copy.Robots[id] = robot
	}
	return copy
}
