package board_test

import (
	"context"
	"errors"
	"fmt"
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
		"p1": state(0, 0, board.East), "p2": state(2, 0, board.West),
	}}, []game.RobotID{"p1", "p2"})
	initial := b.Snapshot()
	d := next(t, b, nil)
	if d.Request == nil || d.Request.RobotID != "p1" || d.Request.ReturnValue != nil {
		t.Errorf("unexpected initial decision: %+v", d)
	}
	query := game.DrillDurabilityAction{}
	for range 3 {
		d = next(t, b, &game.Reply{RobotID: "p1", Action: query})
		if d.Request == nil || d.Request.RobotID != "p1" || d.Request.ReturnValue == nil || *d.Request.ReturnValue != (game.Value{Kind: game.Integer, Integer: 3}) {
			t.Errorf("unexpected query result: %+v", d)
		}
		if !reflect.DeepEqual(initial, b.Snapshot()) {
			t.Error("query changed the board")
		}
	}
	move := game.ForwardAction{}
	d = next(t, b, &game.Reply{RobotID: "p1", Action: move})
	if d.Request == nil || d.Request.RobotID != "p2" {
		t.Errorf("expected p2, got %+v", d)
	}
	d = next(t, b, &game.Reply{RobotID: "p2", Action: move})
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
	forward := game.ForwardAction{}
	tests := []struct {
		name    string
		grid    [][]board.Terrain
		robots  map[game.RobotID]board.RobotState
		ids     []game.RobotID
		actions map[game.RobotID]game.Action
		want    map[game.RobotID]board.RobotState
		moved   map[game.RobotID]bool
	}{
		{
			name:   "chain into empty cell",
			grid:   land(4, 1),
			robots: map[game.RobotID]board.RobotState{"a": state(0, 0, board.East), "b": state(1, 0, board.East)},
			ids:    []game.RobotID{"a", "b"},
			want:   map[game.RobotID]board.RobotState{"a": state(1, 0, board.East), "b": state(2, 0, board.East)},
			moved:  map[game.RobotID]bool{"a": true, "b": true},
		},
		{
			name:   "swap",
			grid:   land(2, 1),
			robots: map[game.RobotID]board.RobotState{"a": state(0, 0, board.East), "b": state(1, 0, board.West)},
			ids:    []game.RobotID{"b", "a"},
		},
		{
			name: "cycle",
			grid: land(2, 2),
			robots: map[game.RobotID]board.RobotState{
				"a": state(0, 0, board.East), "b": state(1, 0, board.South),
				"c": state(1, 1, board.West), "d": state(0, 1, board.North),
			},
			ids: []game.RobotID{"a", "b", "c", "d"},
		},
		{
			name:   "conflict propagates",
			grid:   land(4, 1),
			robots: map[game.RobotID]board.RobotState{"a": state(0, 0, board.East), "b": state(1, 0, board.East), "c": state(3, 0, board.West)},
			ids:    []game.RobotID{"a", "b", "c"},
		},
		{
			name:   "boundary propagates",
			grid:   land(2, 1),
			robots: map[game.RobotID]board.RobotState{"a": state(0, 0, board.East), "b": state(1, 0, board.East)},
			ids:    []game.RobotID{"a", "b"},
		},
		{
			name:   "rock",
			grid:   [][]board.Terrain{{board.Land, board.Rock}},
			robots: map[game.RobotID]board.RobotState{"a": state(0, 0, board.East)},
			ids:    []game.RobotID{"a"},
		},
		{
			name:   "sea",
			grid:   [][]board.Terrain{{board.Land, board.Sea}},
			robots: map[game.RobotID]board.RobotState{"a": state(0, 0, board.East)},
			ids:    []game.RobotID{"a"},
		},
		{
			name:    "back",
			grid:    land(2, 1),
			robots:  map[game.RobotID]board.RobotState{"a": state(1, 0, board.East)},
			ids:     []game.RobotID{"a"},
			actions: map[game.RobotID]game.Action{"a": game.BackAction{}},
			want:    map[game.RobotID]board.RobotState{"a": state(0, 0, board.East)},
			moved:   map[game.RobotID]bool{"a": true},
		},
		{
			name:    "rotate blocks follower",
			grid:    land(2, 1),
			robots:  map[game.RobotID]board.RobotState{"a": state(0, 0, board.East), "b": state(1, 0, board.North)},
			ids:     []game.RobotID{"a", "b"},
			actions: map[game.RobotID]game.Action{"b": game.RotateAction{Rotation: game.Left}},
			want:    map[game.RobotID]board.RobotState{"a": state(0, 0, board.East), "b": state(1, 0, board.West)},
		},
		{
			name:    "rotate right wraps",
			grid:    land(1, 1),
			robots:  map[game.RobotID]board.RobotState{"a": state(0, 0, board.West)},
			ids:     []game.RobotID{"a"},
			actions: map[game.RobotID]game.Action{"a": game.RotateAction{Rotation: game.Right}},
			want:    map[game.RobotID]board.RobotState{"a": state(0, 0, board.North)},
		},
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
				var a game.Action = forward
				if custom, ok := tt.actions[id]; ok {
					a = custom
				}
				d = next(t, b, &game.Reply{RobotID: id, Action: a})
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
					if _, rotating := result.Action.(game.RotateAction); rotating {
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
	valid := board.Config{Terrain: land(2, 1), Robots: map[game.RobotID]board.RobotState{"a": state(0, 0, board.East)}}
	b, err := board.New(valid)
	if err != nil {
		t.Fatal(err)
	}
	initial := b.Snapshot()
	valid.Terrain[0][0] = board.Sea
	valid.Robots["a"] = state(1, 0, board.West)
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
		{Terrain: land(1, 1), Robots: map[game.RobotID]board.RobotState{"a": state(1, 0, board.North)}},
		{Terrain: land(1, 1), Robots: map[game.RobotID]board.RobotState{"a": state(0, 0, "invalid")}},
		{Terrain: land(1, 1), Robots: map[game.RobotID]board.RobotState{"": state(0, 0, board.North)}},
		{Terrain: land(1, 1), Robots: map[game.RobotID]board.RobotState{"a": state(0, 0, board.North), "b": state(0, 0, board.South)}},
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
	b := newBoard(t, board.Config{Terrain: land(2, 1), Robots: map[game.RobotID]board.RobotState{"a": state(0, 0, board.East)}}, []game.RobotID{"a"})
	initial := b.Snapshot()
	next(t, b, nil)
	for _, reply := range []*game.Reply{
		nil, {RobotID: "other", Done: true}, {RobotID: "a"},
		{RobotID: "a", Action: game.RotateAction{}},
		{RobotID: "a", Action: game.RotateAction{Rotation: "sideways"}},
		{RobotID: "a", Done: true, Err: errors.New("error")},
	} {
		if _, err := b.Next(t.Context(), reply); err == nil {
			t.Error("invalid reply accepted")
		}
		if !reflect.DeepEqual(initial, b.Snapshot()) {
			t.Error("invalid reply changed board")
		}
	}
	next(t, b, &game.Reply{RobotID: "a", Action: game.ForwardAction{}})
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
	b := newBoard(t, board.Config{Terrain: land(2, 1), Robots: map[game.RobotID]board.RobotState{"a": state(0, 0, board.East), "b": state(1, 0, board.East)}}, []game.RobotID{"a", "b"})
	next(t, b, nil)
	next(t, b, &game.Reply{RobotID: "a", Action: game.ForwardAction{}})
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
	field := game.FieldValue{Terrain: game.Land}
	for _, action := range []game.Action{
		game.WaitAction{Integer: 7},
		game.PickUpAction{}, game.ReleaseAction{}, game.DrillAction{},
		game.DistanceAction{}, game.SearchFrontAction{}, game.SearchNowAction{},
		game.IsLandAction{Field: field}, game.IsGoalAction{Field: field},
		game.IsCheckpointAction{Field: field}, game.IsLaggageAction{Field: field},
		game.IsSeaAction{Field: field}, game.IsRockAction{Field: field},
		game.PrintIntegerAction{Integer: -42}, game.PrintBoolAction{Bool: true},
	} {
		t.Run(fmt.Sprintf("%T", action), func(t *testing.T) {
			t.Parallel()
			b := newBoard(t, board.Config{Terrain: land(3, 1), Robots: map[game.RobotID]board.RobotState{
				"a": state(0, 0, board.East), "b": state(2, 0, board.West),
			}}, []game.RobotID{"a", "b"})
			initial := b.Snapshot()
			next(t, b, nil)
			// A valid action is already pending when the other robot sends an
			// unimplemented action; neither action may mutate the board.
			next(t, b, &game.Reply{RobotID: "a", Action: game.ForwardAction{}})
			_, err := b.Next(t.Context(), &game.Reply{RobotID: "b", Action: action})
			if !errors.Is(err, game.ErrNotSupported) {
				t.Errorf("got %v, want ErrNotSupported", err)
			}
			if !reflect.DeepEqual(initial, b.Snapshot()) {
				t.Error("unsupported action committed pending changes")
			}
		})
	}
}
