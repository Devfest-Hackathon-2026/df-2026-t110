// Package board provides a deterministic grid board and logical-tick scheduler.
package board

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/dvpico/Programming-Learning-AR/Server/game"
)

type Terrain string

const (
	Land Terrain = "land"
	Rock Terrain = "rock"
	Sea  Terrain = "sea"
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

type RobotState struct {
	Position        Position
	Direction       Direction
	DrillDurability int64
}

type Config struct {
	Terrain [][]Terrain
	Robots  map[game.RobotID]RobotState
}

type Snapshot struct {
	Tick    uint64
	Terrain [][]Terrain
	Robots  map[game.RobotID]RobotState
}

type control struct {
	done    bool
	pending *game.Action
	value   *game.Value
}

type Board struct {
	state    Snapshot
	order    []game.RobotID
	controls map[game.RobotID]*control
	started  bool
	awaiting game.RobotID
	advance  bool
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
			if cell != Land && cell != Rock && cell != Sea {
				return nil, fmt.Errorf("unknown terrain %q", cell)
			}
		}
	}
	occupied := make(map[Position]bool)
	for id, robot := range config.Robots {
		p := robot.Position
		if id == "" || directionIndex(robot.Direction) < 0 || robot.DrillDurability < 0 ||
			p.X < 0 || p.Y < 0 || p.X >= width || p.Y >= len(config.Terrain) {
			return nil, fmt.Errorf("invalid robot %q", id)
		}
		if config.Terrain[p.Y][p.X] != Land || occupied[p] {
			return nil, fmt.Errorf("robot %q must occupy a unique land cell", id)
		}
		occupied[p] = true
	}
	state := Snapshot{Terrain: config.Terrain, Robots: config.Robots}
	return &Board{state: clone(state)}, nil
}

func (b *Board) Snapshot() Snapshot { return clone(b.state) }

func (b *Board) Start(ctx context.Context, ids []game.RobotID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if b.started || len(ids) != len(b.state.Robots) {
		return errors.New("board is single-use and robot IDs must match its configuration")
	}
	controls := make(map[game.RobotID]*control, len(ids))
	for _, id := range ids {
		if _, ok := b.state.Robots[id]; !ok || controls[id] != nil {
			return fmt.Errorf("unknown or duplicate robot %q", id)
		}
		controls[id] = &control{}
	}
	b.controls, b.order, b.started = controls, append([]game.RobotID(nil), ids...), true
	return nil
}

func (b *Board) Next(ctx context.Context, reply *game.Reply) (game.Decision, error) {
	if err := ctx.Err(); err != nil {
		return game.Decision{}, err
	}
	if !b.started {
		return game.Decision{}, errors.New("board has not started")
	}
	if b.awaiting != "" {
		if err := b.accept(reply); err != nil {
			return game.Decision{}, err
		}
	} else if reply != nil {
		return game.Decision{}, errors.New("unexpected robot reply")
	}
	if b.advance {
		update, err := b.commit(ctx)
		if err != nil {
			return game.Decision{}, err
		}
		b.advance = false
		return game.Decision{Update: &update}, nil
	}
	pending := false
	for _, id := range b.order {
		c := b.controls[id]
		if c.pending != nil {
			pending = true
		} else if !c.done {
			b.awaiting = id
			request := &game.Request{RobotID: id, ReturnValue: c.value}
			c.value = nil
			return game.Decision{Request: request}, nil
		}
	}
	if pending {
		if b.state.Tick == math.MaxUint64 {
			return game.Decision{}, errors.New("tick overflow")
		}
		tick := b.state.Tick + 1
		b.advance = true
		return game.Decision{AdvanceTo: &tick}, nil
	}
	return game.Decision{Done: true}, nil
}

func (b *Board) accept(reply *game.Reply) error {
	if reply == nil || reply.RobotID != b.awaiting {
		return errors.New("reply must match the requested robot")
	}
	count := 0
	for _, set := range []bool{reply.Action != nil, reply.Done, reply.Err != nil} {
		if set {
			count++
		}
	}
	if count != 1 {
		return errors.New("reply must contain an action, completion, or error")
	}
	c := b.controls[b.awaiting]
	if reply.Action != nil {
		a := *reply.Action
		if a.Kind != game.Rotate && a.Rotation != "" {
			return errors.New("only rotate accepts a rotation")
		}
		switch a.Kind {
		case game.DrillDurability:
			c.value = &game.Value{Kind: game.Integer, Integer: b.state.Robots[b.awaiting].DrillDurability}
		case game.Forward, game.Back:
			c.pending = &a
		case game.Rotate:
			if a.Rotation != game.Left && a.Rotation != game.Right {
				return errors.New("rotate requires left or right")
			}
			c.pending = &a
		default:
			return fmt.Errorf("%w: action %q", game.ErrNotSupported, a.Kind)
		}
	} else {
		c.done = true
	}
	b.awaiting = ""
	return nil
}

// commit resolves movement dependencies against one immutable starting state.
// A visiting dependency denotes a swap/cycle; failure propagates to followers.
func (b *Board) commit(ctx context.Context) (game.TickUpdate, error) {
	occupied := make(map[Position]game.RobotID, len(b.state.Robots))
	targets := make(map[game.RobotID]Position)
	counts := make(map[Position]int)
	for id, robot := range b.state.Robots {
		occupied[robot.Position] = id
	}
	for _, id := range b.order {
		a := b.controls[id].pending
		if a == nil || (a.Kind != game.Forward && a.Kind != game.Back) {
			continue
		}
		robot := b.state.Robots[id]
		delta := directionDeltas[robot.Direction]
		if a.Kind == game.Back {
			delta.X, delta.Y = -delta.X, -delta.Y
		}
		target := Position{robot.Position.X + delta.X, robot.Position.Y + delta.Y}
		if b.land(target) {
			targets[id] = target
			counts[target]++
		}
	}
	visiting := make(map[game.RobotID]bool)
	resolved := make(map[game.RobotID]bool)
	movable := make(map[game.RobotID]bool)
	var canMove func(game.RobotID) bool
	canMove = func(id game.RobotID) bool {
		if resolved[id] {
			return movable[id]
		}
		if visiting[id] {
			return false
		}
		target, ok := targets[id]
		if !ok || counts[target] != 1 {
			resolved[id] = true
			return false
		}
		visiting[id] = true
		other, occupied := occupied[target]
		movable[id] = !occupied || canMove(other)
		visiting[id], resolved[id] = false, true
		return movable[id]
	}
	next := clone(b.state)
	next.Tick++
	update := game.TickUpdate{Tick: next.Tick}
	for _, id := range b.order {
		a := b.controls[id].pending
		if a == nil {
			continue
		}
		robot := next.Robots[id]
		value := game.Value{Kind: game.Unit}
		if a.Kind == game.Rotate {
			rotation := 1
			if a.Rotation == game.Left {
				rotation = 3
			}
			robot.Direction = [...]Direction{North, East, South, West}[(directionIndex(robot.Direction)+rotation)%4]
		} else {
			value = game.Value{Kind: game.Bool, Bool: canMove(id)}
			if value.Bool {
				robot.Position = targets[id]
			}
		}
		next.Robots[id] = robot
		update.Results = append(update.Results, game.ActionResult{RobotID: id, Action: *a, Value: value})
	}
	if err := ctx.Err(); err != nil {
		return game.TickUpdate{}, err
	}
	b.state = next
	for _, result := range update.Results {
		c := b.controls[result.RobotID]
		value := result.Value
		c.pending, c.value = nil, &value
	}
	return update, nil
}

func (b *Board) land(p Position) bool {
	return p.Y >= 0 && p.Y < len(b.state.Terrain) && p.X >= 0 &&
		p.X < len(b.state.Terrain[0]) && b.state.Terrain[p.Y][p.X] == Land
}

func directionIndex(d Direction) int {
	for i, direction := range [...]Direction{North, East, South, West} {
		if d == direction {
			return i
		}
	}
	return -1
}

func clone(state Snapshot) Snapshot {
	copy := Snapshot{Tick: state.Tick, Terrain: make([][]Terrain, len(state.Terrain)), Robots: make(map[game.RobotID]RobotState, len(state.Robots))}
	for y, row := range state.Terrain {
		copy.Terrain[y] = append([]Terrain(nil), row...)
	}
	for id, robot := range state.Robots {
		copy.Robots[id] = robot
	}
	return copy
}
