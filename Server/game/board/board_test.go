package board_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game"
	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
)

func land(width, height int) [][]board.Terrain {
	grid := make([][]board.Terrain, height)
	for y := range grid {
		grid[y] = make([]board.Terrain, width)
		for x := range grid[y] {
			grid[y][x] = board.Land
		}
	}
	return grid
}

func state(x, y int, direction board.Direction) board.RobotState {
	return board.RobotState{Position: board.Position{X: x, Y: y}, Direction: direction, DrillDurability: 3}
}

func newBoard(t *testing.T, config board.Config, ids []game.RobotID) *board.Board {
	t.Helper()
	b, err := board.New(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Start(t.Context(), ids); err != nil {
		t.Fatal(err)
	}
	return b
}

func next(t *testing.T, b *board.Board, reply *game.Reply) game.Decision {
	t.Helper()
	d, err := b.Next(t.Context(), reply)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestBarrierAndImmutableQueries(t *testing.T) {
	t.Parallel()
	b := newBoard(t, board.Config{Terrain: land(3, 1), Robots: map[game.RobotID]board.RobotState{
		"p1": state(0, 0, board.Right), "p2": state(2, 0, board.Left),
	}}, []game.RobotID{"p1", "p2"})
	initial := b.Snapshot()
	d := next(t, b, nil)
	if d.Request == nil || d.Request.RobotID != "p1" || d.Request.ReturnValue != nil {
		t.Errorf("unexpected initial decision: %+v", d)
	}
	query := game.Action{Kind: game.DrillDurability}
	for range 3 {
		d = next(t, b, &game.Reply{RobotID: "p1", Action: &query})
		if d.Request == nil || d.Request.RobotID != "p1" || d.Request.ReturnValue == nil || *d.Request.ReturnValue != (game.Value{Kind: game.Integer, Integer: 3}) {
			t.Errorf("unexpected query result: %+v", d)
		}
		if !reflect.DeepEqual(initial, b.Snapshot()) {
			t.Error("query changed the board")
		}
	}
	move := game.Action{Kind: game.Forward}
	d = next(t, b, &game.Reply{RobotID: "p1", Action: &move})
	if d.Request == nil || d.Request.RobotID != "p2" {
		t.Errorf("expected p2, got %+v", d)
	}
	d = next(t, b, &game.Reply{RobotID: "p2", Action: &move})
	if d.AdvanceTo == nil || *d.AdvanceTo != 1 || !reflect.DeepEqual(initial, b.Snapshot()) {
		t.Error("board must wait for commit authorization")
	}
	d = next(t, b, nil)
	if d.Update == nil || d.Update.Tick != 1 || len(d.Update.Results) != 2 {
		t.Errorf("unexpected commit: %+v", d)
	}
	if d.Update != nil {
		for _, result := range d.Update.Results {
			if result.Value.Kind != game.Bool || result.Value.Bool {
				t.Error("same destination must reject both movements")
			}
		}
	}
	d = next(t, b, nil)
	if d.Request == nil || d.Request.RobotID != "p1" || d.Request.ReturnValue == nil || d.Request.ReturnValue.Kind != game.Bool || d.Request.ReturnValue.Bool {
		t.Error("move result was not delivered")
	}
}

func TestAtomicMovement(t *testing.T) {
	t.Parallel()
	forward := game.Action{Kind: game.Forward}
	tests := []struct {
		name    string
		grid    [][]board.Terrain
		robots  map[game.RobotID]board.RobotState
		ids     []game.RobotID
		actions map[game.RobotID]game.Action
		want    map[game.RobotID]board.RobotState
		moved   map[game.RobotID]bool
	}{
		{"chain into empty cell", land(4, 1), map[game.RobotID]board.RobotState{"a": state(0, 0, board.Right), "b": state(1, 0, board.Right)}, []game.RobotID{"a", "b"}, nil,
			map[game.RobotID]board.RobotState{"a": state(1, 0, board.Right), "b": state(2, 0, board.Right)}, map[game.RobotID]bool{"a": true, "b": true}},
		{"swap", land(2, 1), map[game.RobotID]board.RobotState{"a": state(0, 0, board.Right), "b": state(1, 0, board.Left)}, []game.RobotID{"b", "a"}, nil, nil, nil},
		{"cycle", land(2, 2), map[game.RobotID]board.RobotState{"a": state(0, 0, board.Right), "b": state(1, 0, board.Down), "c": state(1, 1, board.Left), "d": state(0, 1, board.Up)}, []game.RobotID{"a", "b", "c", "d"}, nil, nil, nil},
		{"conflict propagates", land(4, 1), map[game.RobotID]board.RobotState{"a": state(0, 0, board.Right), "b": state(1, 0, board.Right), "c": state(3, 0, board.Left)}, []game.RobotID{"a", "b", "c"}, nil, nil, nil},
		{"boundary propagates", land(2, 1), map[game.RobotID]board.RobotState{"a": state(0, 0, board.Right), "b": state(1, 0, board.Right)}, []game.RobotID{"a", "b"}, nil, nil, nil},
		{"rock", [][]board.Terrain{{board.Land, board.Rock}}, map[game.RobotID]board.RobotState{"a": state(0, 0, board.Right)}, []game.RobotID{"a"}, nil, nil, nil},
		{"sea", [][]board.Terrain{{board.Land, board.Sea}}, map[game.RobotID]board.RobotState{"a": state(0, 0, board.Right)}, []game.RobotID{"a"}, nil, nil, nil},
		{"back", land(2, 1), map[game.RobotID]board.RobotState{"a": state(1, 0, board.Right)}, []game.RobotID{"a"}, map[game.RobotID]game.Action{"a": {Kind: game.Back}},
			map[game.RobotID]board.RobotState{"a": state(0, 0, board.Right)}, map[game.RobotID]bool{"a": true}},
		{"rotate blocks follower", land(2, 1), map[game.RobotID]board.RobotState{"a": state(0, 0, board.Right), "b": state(1, 0, board.Up)}, []game.RobotID{"a", "b"}, map[game.RobotID]game.Action{"b": {Kind: game.Rotate, Turn: game.Left}},
			map[game.RobotID]board.RobotState{"a": state(0, 0, board.Right), "b": state(1, 0, board.Left)}, nil},
		{"rotate right wraps", land(1, 1), map[game.RobotID]board.RobotState{"a": state(0, 0, board.Left)}, []game.RobotID{"a"}, map[game.RobotID]game.Action{"a": {Kind: game.Rotate, Turn: game.Right}},
			map[game.RobotID]board.RobotState{"a": state(0, 0, board.Up)}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := newBoard(t, board.Config{Terrain: tt.grid, Robots: tt.robots}, tt.ids)
			d := next(t, b, nil)
			for _, id := range tt.ids {
				if d.Request == nil || d.Request.RobotID != id {
					t.Errorf("unexpected request: %+v", d)
				}
				a := forward
				if custom, ok := tt.actions[id]; ok {
					a = custom
				}
				d = next(t, b, &game.Reply{RobotID: id, Action: &a})
			}
			if d.AdvanceTo == nil {
				t.Error("missing advance proposal")
			}
			d = next(t, b, nil)
			if d.Update == nil || b.Snapshot().Tick != 1 {
				t.Error("missing tick commit")
			}
			want := tt.want
			if want == nil {
				want = tt.robots
			}
			if !reflect.DeepEqual(b.Snapshot().Robots, want) {
				t.Errorf("got %+v, want %+v", b.Snapshot().Robots, want)
			}
			if d.Update != nil {
				for _, result := range d.Update.Results {
					if result.Action.Kind == game.Rotate {
						if result.Value.Kind != game.Unit {
							t.Error("rotation must return unit")
						}
					} else if result.Value.Kind != game.Bool || result.Value.Bool != tt.moved[result.RobotID] {
						t.Errorf("unexpected movement result: %+v", result)
					}
				}
			}
		})
	}
}

func TestValidationAndCopies(t *testing.T) {
	t.Parallel()
	valid := board.Config{Terrain: land(2, 1), Robots: map[game.RobotID]board.RobotState{"a": state(0, 0, board.Right)}}
	b, err := board.New(valid)
	if err != nil {
		t.Fatal(err)
	}
	initial := b.Snapshot()
	valid.Terrain[0][0] = board.Sea
	valid.Robots["a"] = state(1, 0, board.Left)
	copy := b.Snapshot()
	copy.Terrain[0][0] = board.Rock
	delete(copy.Robots, "a")
	if !reflect.DeepEqual(b.Snapshot(), initial) {
		t.Error("constructor or snapshot aliases caller data")
	}
	invalid := []board.Config{
		{}, {Terrain: land(2, 1)},
		{Terrain: [][]board.Terrain{{board.Land}, {}}, Robots: initial.Robots},
		{Terrain: [][]board.Terrain{{"unknown"}}, Robots: initial.Robots},
		{Terrain: [][]board.Terrain{{board.Sea}}, Robots: initial.Robots},
		{Terrain: land(1, 1), Robots: map[game.RobotID]board.RobotState{"a": state(1, 0, board.Up)}},
		{Terrain: land(1, 1), Robots: map[game.RobotID]board.RobotState{"a": state(0, 0, "invalid")}},
		{Terrain: land(1, 1), Robots: map[game.RobotID]board.RobotState{"": state(0, 0, board.Up)}},
		{Terrain: land(1, 1), Robots: map[game.RobotID]board.RobotState{"a": state(0, 0, board.Up), "b": state(0, 0, board.Down)}},
	}
	for i, config := range invalid {
		if _, err := board.New(config); err == nil {
			t.Errorf("invalid config %d accepted", i)
		}
	}
	if err := b.Start(t.Context(), []game.RobotID{"missing"}); err == nil {
		t.Error("unknown robot accepted")
	}
	if err := b.Start(t.Context(), []game.RobotID{"a"}); err != nil {
		t.Error(err)
	}
	if err := b.Start(t.Context(), []game.RobotID{"a"}); err == nil {
		t.Error("second start accepted")
	}
}

func TestRejectedRepliesAndCancellationDoNotCommit(t *testing.T) {
	t.Parallel()
	b := newBoard(t, board.Config{Terrain: land(2, 1), Robots: map[game.RobotID]board.RobotState{"a": state(0, 0, board.Right)}}, []game.RobotID{"a"})
	initial := b.Snapshot()
	next(t, b, nil)
	for _, reply := range []*game.Reply{
		nil, {RobotID: "other", Done: true}, {RobotID: "a"},
		{RobotID: "a", Action: &game.Action{Kind: game.Rotate}},
		{RobotID: "a", Action: &game.Action{Kind: "unknown"}},
		{RobotID: "a", Action: &game.Action{Kind: game.Forward, Turn: game.Left}},
		{RobotID: "a", Done: true, Err: errors.New("error")},
	} {
		if _, err := b.Next(t.Context(), reply); err == nil {
			t.Error("invalid reply accepted")
		}
		if !reflect.DeepEqual(initial, b.Snapshot()) {
			t.Error("invalid reply changed board")
		}
	}
	next(t, b, &game.Reply{RobotID: "a", Action: &game.Action{Kind: game.Forward}})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := b.Next(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want cancellation", err)
	}
	if !reflect.DeepEqual(initial, b.Snapshot()) {
		t.Error("cancellation committed pending action")
	}
}

func TestFinishedRobotRemainsAnObstacle(t *testing.T) {
	t.Parallel()
	b := newBoard(t, board.Config{Terrain: land(2, 1), Robots: map[game.RobotID]board.RobotState{"a": state(0, 0, board.Right), "b": state(1, 0, board.Right)}}, []game.RobotID{"a", "b"})
	next(t, b, nil)
	next(t, b, &game.Reply{RobotID: "a", Action: &game.Action{Kind: game.Forward}})
	next(t, b, &game.Reply{RobotID: "b", Done: true})
	d := next(t, b, nil)
	if d.Update == nil || len(d.Update.Results) != 1 || d.Update.Results[0].Value.Bool {
		t.Error("finished robot did not block movement")
	}
	next(t, b, nil)
	if !next(t, b, &game.Reply{RobotID: "a", Done: true}).Done {
		t.Error("board did not finish")
	}
}

func TestUnimplementedActionsAreNotSupported(t *testing.T) {
	t.Parallel()
	for _, action := range []game.Action{
		{Kind: game.Wait, Integer: 7},
		{Kind: game.PickUp}, {Kind: game.Release}, {Kind: game.Drill},
		{Kind: game.Distance}, {Kind: game.SearchFront}, {Kind: game.SearchNow},
		{Kind: game.IsLand, Field: "observation"}, {Kind: game.IsGoal, Field: "observation"},
		{Kind: game.IsCheckpoint, Field: "observation"}, {Kind: game.IsLaggage, Field: "observation"},
		{Kind: game.IsSea, Field: "observation"}, {Kind: game.IsRock, Field: "observation"},
		{Kind: game.PrintInteger, Integer: -42}, {Kind: game.PrintBool, Bool: true},
	} {
		t.Run(string(action.Kind), func(t *testing.T) {
			t.Parallel()
			b := newBoard(t, board.Config{Terrain: land(3, 1), Robots: map[game.RobotID]board.RobotState{
				"a": state(0, 0, board.Right), "b": state(2, 0, board.Left),
			}}, []game.RobotID{"a", "b"})
			initial := b.Snapshot()
			next(t, b, nil)
			// A valid action is already pending when the other robot sends an
			// unimplemented action; neither action may mutate the board.
			next(t, b, &game.Reply{RobotID: "a", Action: &game.Action{Kind: game.Forward}})
			_, err := b.Next(t.Context(), &game.Reply{RobotID: "b", Action: &action})
			if !errors.Is(err, game.ErrNotSupported) {
				t.Errorf("got %v, want ErrNotSupported", err)
			}
			if !reflect.DeepEqual(initial, b.Snapshot()) {
				t.Error("unsupported action committed pending changes")
			}
		})
	}
}
