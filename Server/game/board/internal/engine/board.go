// Package engine implements the grid board and its logical-tick scheduler.
package engine

import (
	"errors"
	"fmt"
	"math"

	"github.com/dvpico/Programming-Learning-AR/Server/game"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

var ErrTickLimit = errors.New("tick limit reached")

type Direction string

const (
	North Direction = "north"
	East  Direction = "east"
	South Direction = "south"
	West  Direction = "west"
)

type Position struct{ x, y int }

func NewPosition(x, y int) Position { return Position{x: x, y: y} }

func (p Position) X() int { return p.x }
func (p Position) Y() int { return p.y }

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
	position        Position
	direction       Direction
	drillDurability int64
}

func NewRobotState(position Position, direction Direction, drillDurability int64) RobotState {
	return RobotState{position: position, direction: direction, drillDurability: drillDurability}
}

func (r RobotState) Position() Position     { return r.position }
func (r RobotState) Direction() Direction   { return r.direction }
func (r RobotState) DrillDurability() int64 { return r.drillDurability }

type Config struct {
	// MaxTicks bounds committed ticks; zero means unlimited.
	MaxTicks uint64
	Terrain  [][]Terrain
	Robots   map[protocol.RobotID]RobotState
}

type Snapshot struct {
	Tick    uint64
	Terrain [][]Terrain
	Robots  map[protocol.RobotID]RobotState
}

type Board struct {
	maxTicks uint64
	state    Snapshot
	order    []protocol.RobotID
	controls map[protocol.RobotID]control
	phase    phase
}

func New(config Config) (Board, error) {
	if len(config.Terrain) == 0 || len(config.Terrain[0]) == 0 || len(config.Robots) == 0 {
		return Board{}, errors.New("nonempty terrain and robots are required")
	}
	width := len(config.Terrain[0])
	for _, row := range config.Terrain {
		if len(row) != width {
			return Board{}, errors.New("terrain must be rectangular")
		}
		for _, cell := range row {
			if !validTerrain(cell) {
				return Board{}, fmt.Errorf("invalid terrain %T", cell)
			}
		}
	}
	occupied := make(map[Position]struct{}, len(config.Robots))
	for id, robot := range config.Robots {
		p := robot.position
		if id == "" || directionIndex(robot.direction) < 0 || robot.drillDurability < 0 ||
			p.x < 0 || p.y < 0 || p.x >= width || p.y >= len(config.Terrain) {
			return Board{}, fmt.Errorf("invalid robot %q", id)
		}
		_, isLand := config.Terrain[p.y][p.x].(LandTerrain)
		if _, exists := occupied[p]; !isLand || exists {
			return Board{}, fmt.Errorf("robot %q must occupy a unique land cell", id)
		}
		occupied[p] = struct{}{}
	}
	state := Snapshot{Terrain: config.Terrain, Robots: config.Robots}
	return Board{maxTicks: config.MaxTicks, state: clone(state), phase: unstartedBoard{}}, nil
}

func (b *Board) Snapshot() Snapshot { return clone(b.state) }

func (b *Board) Start(ids []protocol.RobotID) ([]fieldevent.FieldEvent, error) {
	if _, fresh := b.phase.(unstartedBoard); !fresh || len(ids) != len(b.state.Robots) {
		return nil, errors.New("board is single-use and robot IDs must match its configuration")
	}
	controls := make(map[protocol.RobotID]control, len(ids))
	for _, id := range ids {
		if _, ok := b.state.Robots[id]; !ok || controls[id] != nil {
			return nil, fmt.Errorf("unknown or duplicate robot %q", id)
		}
		controls[id] = unstartedControl{}
	}
	b.controls, b.order, b.phase = controls, append([]protocol.RobotID(nil), ids...), collectingBoard{}
	return []fieldevent.FieldEvent{{State: b.initialFieldState()}}, nil
}

func (b *Board) Next(reply robotmsg.Reply) (boardmsg.BoardOutput, error) {
	switch phase := b.phase.(type) {
	case unstartedBoard:
		return boardmsg.BoardOutput{}, errors.New("board has not started")
	case awaitingReplyBoard:
		if err := b.accept(phase.robotID, reply); err != nil {
			return boardmsg.BoardOutput{}, err
		}
		b.phase = collectingBoard{}
	case collectingBoard, finishedBoard:
		if reply != nil {
			return boardmsg.BoardOutput{}, errors.New("unexpected robot reply")
		}
	default:
		return boardmsg.BoardOutput{}, errors.New("invalid board phase")
	}

	var output boardmsg.BoardOutput
	for {
		if _, finished := b.phase.(finishedBoard); finished {
			output.Decision = boardmsg.DoneDecision{}
			return output, nil
		}

		hasPendingActions := false
		for _, id := range b.order {
			switch c := b.controls[id].(type) {
			case pendingControl:
				hasPendingActions = true
			case unstartedControl:
				b.phase = awaitingReplyBoard{robotID: id}
				output.Decision = boardmsg.StartDecision{RobotID: id}
				return output, nil
			case readyControl:
				b.phase = awaitingReplyBoard{robotID: id}
				output.Decision = boardmsg.ResumeDecision{RobotID: id, ReturnValue: c.value}
				return output, nil
			case finishedControl:
			default:
				return output, fmt.Errorf("invalid control for robot %q", id)
			}
		}

		if !hasPendingActions {
			// All controls are finished here. Record the remaining robots' removal
			// before clearing the board, at the current tick.
			var update fieldevent.UpdateState
			for _, id := range b.order {
				if _, present := b.state.Robots[id]; present {
					update.RemovedRobotIDs = append(update.RemovedRobotIDs, id)
				}
			}
			clear(b.state.Robots)
			b.phase = finishedBoard{}
			output.Events = append(output.Events, fieldevent.FieldEvent{Tick: b.state.Tick, State: update})
			output.Decision = boardmsg.DoneDecision{}
			return output, nil
		}

		if b.state.Tick == math.MaxUint64 {
			return output, errors.New("tick overflow")
		}
		if b.maxTicks != 0 && b.state.Tick >= b.maxTicks {
			return output, ErrTickLimit
		}
		update, err := b.commit()
		if err != nil {
			return output, err
		}
		output.Events = append(output.Events, fieldevent.FieldEvent{Tick: b.state.Tick, State: update})
	}
}

func (b *Board) accept(id protocol.RobotID, reply robotmsg.Reply) error {
	var robotID protocol.RobotID
	switch r := reply.(type) {
	case robotmsg.ActionReply:
		robotID = r.RobotID
	case robotmsg.DoneReply:
		robotID = r.RobotID
	case robotmsg.ErrorReply:
		robotID = r.RobotID
	default:
		return errors.New("reply must contain an action, completion, or error")
	}
	if robotID != id {
		return errors.New("reply must match the requested robot")
	}

	switch r := reply.(type) {
	case robotmsg.ActionReply:
		if r.Action == nil {
			return errors.New("action reply must contain an action")
		}
		switch a := r.Action.(type) {
		case actions.DrillDurabilityAction:
			b.controls[id] = readyControl{value: values.IntegerValue{Integer: b.state.Robots[id].drillDurability}}
		case actions.ForwardAction, actions.BackAction:
			b.controls[id] = pendingControl{action: a}
		case actions.RotateAction:
			if a.Rotation != values.Left && a.Rotation != values.Right {
				return errors.New("rotate requires left or right")
			}
			b.controls[id] = pendingControl{action: a}
		default:
			return fmt.Errorf("%w: action %T", game.ErrNotSupported, a)
		}
	case robotmsg.DoneReply:
		b.controls[id] = finishedControl{}
	case robotmsg.ErrorReply:
		if r.Err == nil {
			return errors.New("error reply must contain an error")
		}
		b.controls[id] = finishedControl{}
	}
	return nil
}

// commit resolves movement dependencies against one immutable starting state.
func (b *Board) commit() (fieldevent.UpdateState, error) {
	robotCount := len(b.state.Robots)
	occupied := make(map[Position]protocol.RobotID, robotCount)
	targets := make(map[protocol.RobotID]Position, robotCount)
	counts := make(map[Position]int, robotCount)
	next := clone(b.state)
	next.Tick++

	var update fieldevent.UpdateState
	for _, id := range b.order {
		robot, present := b.state.Robots[id]
		if !present {
			continue
		}
		if _, stopped := b.controls[id].(finishedControl); stopped {
			delete(next.Robots, id)
			update.RemovedRobotIDs = append(update.RemovedRobotIDs, id)
			continue
		}
		occupied[robot.position] = id
	}

	for _, id := range b.order {
		c, pending := b.controls[id].(pendingControl)
		if !pending {
			continue
		}
		a := c.action
		robot := b.state.Robots[id]
		delta := directionDeltas[robot.direction]
		switch a.(type) {
		case actions.ForwardAction:
		case actions.BackAction:
			delta.x, delta.y = -delta.x, -delta.y
		default:
			continue
		}
		target := Position{x: robot.position.x + delta.x, y: robot.position.y + delta.y}
		if b.land(target) {
			targets[id] = target
			counts[target]++
		}
	}

	visiting := make(map[protocol.RobotID]struct{}, robotCount)
	resolved := make(map[protocol.RobotID]struct{}, robotCount)
	movable := make(map[protocol.RobotID]bool, robotCount)
	var canMove func(protocol.RobotID) bool
	canMove = func(id protocol.RobotID) bool {
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
		if reverse, moving := targets[other]; occupied && moving && reverse == b.state.Robots[id].position {
			resolved[id] = struct{}{}
			return false
		}
		visiting[id] = struct{}{}
		movable[id] = !occupied || canMove(other)
		delete(visiting, id)
		resolved[id] = struct{}{}
		return movable[id]
	}
	var results []robotmsg.ActionResult
	for _, id := range b.order {
		c, pending := b.controls[id].(pendingControl)
		if !pending {
			continue
		}

		robot := next.Robots[id]
		var value values.Value = values.UnitValue{}
		switch action := c.action.(type) {
		case actions.RotateAction:
			rotation := 1
			if action.Rotation == values.Left {
				rotation = 3
			}
			robot.direction = [...]Direction{North, East, South, West}[(directionIndex(robot.direction)+rotation)%4]
			update.Actions = append(update.Actions, fieldevent.RotateResult{
				RobotID: id, Rotation: action.Rotation, Direction: string(robot.direction),
			})
		case actions.ForwardAction, actions.BackAction:
			moved := canMove(id)
			value = values.BoolValue{Bool: moved}
			kind := fieldevent.Forward
			if _, backward := action.(actions.BackAction); backward {
				kind = fieldevent.Back
			}
			move := fieldevent.MoveResult{Kind: kind, RobotID: id, Success: moved}
			if moved {
				robot.position = targets[id]
				move.To = &fieldevent.Position{X: robot.position.x, Y: robot.position.y}
			}
			update.Actions = append(update.Actions, move)
		default:
			return fieldevent.UpdateState{}, fmt.Errorf("%w: pending action %T", game.ErrNotSupported, c.action)
		}

		next.Robots[id] = robot
		results = append(results, robotmsg.ActionResult{RobotID: id, Action: c.action, Value: value})
	}

	b.state = next
	for _, result := range results {
		b.controls[result.RobotID] = readyControl{value: result.Value}
	}

	return update, nil
}

func (b *Board) land(p Position) bool {
	if p.y < 0 || p.y >= len(b.state.Terrain) || p.x < 0 || p.x >= len(b.state.Terrain[0]) {
		return false
	}
	_, isLand := b.state.Terrain[p.y][p.x].(LandTerrain)
	return isLand
}

func directionIndex(d Direction) int {
	if index, ok := directionIndices[d]; ok {
		return index
	}
	return -1
}

func clone(state Snapshot) Snapshot {
	copy := Snapshot{Tick: state.Tick, Terrain: make([][]Terrain, len(state.Terrain)), Robots: make(map[protocol.RobotID]RobotState, len(state.Robots))}
	for y, row := range state.Terrain {
		copy.Terrain[y] = append([]Terrain(nil), row...)
	}
	for id, robot := range state.Robots {
		copy.Robots[id] = robot
	}
	return copy
}
