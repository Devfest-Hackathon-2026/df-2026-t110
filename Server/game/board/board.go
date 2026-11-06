// Package board provides a deterministic grid board and logical-tick scheduler.
package board

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
	// MaxTicks bounds committed ticks; zero means unlimited.
	MaxTicks uint64
	Terrain  [][]values.TerrainValue
	Robots   map[protocol.RobotID]RobotState
}

type Snapshot struct {
	Tick    uint64
	Terrain [][]values.TerrainValue
	Robots  map[protocol.RobotID]RobotState
}

type Board struct {
	maxTicks uint64
	state    Snapshot
	order    []protocol.RobotID
	controls map[protocol.RobotID]control
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
			if !values.ValidValue(cell) {
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
		_, isLand := config.Terrain[p.Y][p.X].(values.LandTerrain)
		if _, exists := occupied[p]; !isLand || exists {
			return nil, fmt.Errorf("robot %q must occupy a unique land cell", id)
		}
		occupied[p] = struct{}{}
	}
	state := Snapshot{Terrain: config.Terrain, Robots: config.Robots}
	return &Board{maxTicks: config.MaxTicks, state: clone(state), phase: unstartedBoard{}}, nil
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
	return b.events(), nil
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
		pending := false
		for _, id := range b.order {
			switch c := b.controls[id].(type) {
			case pendingControl:
				pending = true
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
		if !pending {
			clear(b.state.Robots)
			b.phase = finishedBoard{}
			output.Decision = boardmsg.DoneDecision{}
			return output, nil
		}
		if b.state.Tick == math.MaxUint64 {
			return output, errors.New("tick overflow")
		}
		if b.maxTicks != 0 && b.state.Tick >= b.maxTicks {
			return output, ErrTickLimit
		}
		if err := b.commit(); err != nil {
			return output, err
		}
		output.Events = append(output.Events, b.events()...)
	}
}

func (b *Board) events() []fieldevent.FieldEvent {
	return []fieldevent.FieldEvent{{Tick: b.state.Tick}}
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
			b.controls[id] = readyControl{value: values.IntegerValue{Integer: b.state.Robots[id].DrillDurability}}
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
func (b *Board) commit() error {
	robotCount := len(b.state.Robots)
	occupied := make(map[Position]protocol.RobotID, robotCount)
	targets := make(map[protocol.RobotID]Position, robotCount)
	counts := make(map[Position]int, robotCount)
	next := clone(b.state)
	next.Tick++
	for id, robot := range b.state.Robots {
		if _, stopped := b.controls[id].(finishedControl); stopped {
			delete(next.Robots, id)
			continue
		}
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
		case actions.ForwardAction:
		case actions.BackAction:
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
	var results []robotmsg.ActionResult
	for _, id := range b.order {
		c, pending := b.controls[id].(pendingControl)
		if !pending {
			continue
		}
		a := c.action
		robot := next.Robots[id]
		var value values.Value = values.UnitValue{}
		switch action := a.(type) {
		case actions.RotateAction:
			rotation := 1
			if action.Rotation == values.Left {
				rotation = 3
			}
			robot.Direction = [...]Direction{North, East, South, West}[(directionIndex(robot.Direction)+rotation)%4]
		case actions.ForwardAction, actions.BackAction:
			moved := canMove(id)
			value = values.BoolValue{Bool: moved}
			if moved {
				robot.Position = targets[id]
			}
		default:
			return fmt.Errorf("%w: pending action %T", game.ErrNotSupported, a)
		}
		next.Robots[id] = robot
		results = append(results, robotmsg.ActionResult{RobotID: id, Action: a, Value: value})
	}
	b.state = next
	for _, result := range results {
		b.controls[result.RobotID] = readyControl{value: result.Value}
	}
	return nil
}

func (b *Board) land(p Position) bool {
	if p.Y < 0 || p.Y >= len(b.state.Terrain) || p.X < 0 || p.X >= len(b.state.Terrain[0]) {
		return false
	}
	_, isLand := b.state.Terrain[p.Y][p.X].(values.LandTerrain)
	return isLand
}

func directionIndex(d Direction) int {
	if index, ok := directionIndices[d]; ok {
		return index
	}
	return -1
}

func clone(state Snapshot) Snapshot {
	copy := Snapshot{Tick: state.Tick, Terrain: make([][]values.TerrainValue, len(state.Terrain)), Robots: make(map[protocol.RobotID]RobotState, len(state.Robots))}
	for y, row := range state.Terrain {
		copy.Terrain[y] = append([]values.TerrainValue(nil), row...)
	}
	for id, robot := range state.Robots {
		copy.Robots[id] = robot
	}
	return copy
}
