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

func land(width, height int) [][]game.TerrainValue {
	grid := make([][]game.TerrainValue, height)
	for y := range grid {
		grid[y] = make([]game.TerrainValue, width)
		for x := range grid[y] {
			grid[y][x] = game.LandTerrain{Kind: game.PlainLand{}}
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

func next(t *testing.T, b *board.Board, reply game.Reply) game.Decision {
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
	if d != (game.StartDecision{RobotID: "p1"}) {
		t.Errorf("unexpected initial decision: %+v", d)
	}
	query := game.DrillDurabilityAction{}
	for range 3 {
		d = next(t, b, game.ActionReply{RobotID: "p1", Action: query})
		if d != (game.ResumeDecision{RobotID: "p1", ReturnValue: game.IntegerValue{Integer: 3}}) {
			t.Errorf("unexpected query result: %+v", d)
		}
		if !reflect.DeepEqual(initial, b.Snapshot()) {
			t.Error("query changed the board")
		}
	}
	move := game.ForwardAction{}
	d = next(t, b, game.ActionReply{RobotID: "p1", Action: move})
	if d != (game.StartDecision{RobotID: "p2"}) {
		t.Errorf("expected p2, got %+v", d)
	}
	d = next(t, b, game.ActionReply{RobotID: "p2", Action: move})
	if d != (game.AdvanceDecision{Tick: 1}) || !reflect.DeepEqual(initial, b.Snapshot()) {
		t.Error("board must wait for commit authorization")
	}
	d = next(t, b, nil)
	update, committed := d.(game.UpdateDecision)
	if !committed || update.Update.Tick != 1 || len(update.Update.Results) != 2 {
		t.Errorf("unexpected commit: %+v", d)
	}
	for _, result := range update.Update.Results {
		if result.Value != (game.BoolValue{Bool: false}) {
			t.Error("same destination must reject both movements")
		}
	}
	d = next(t, b, nil)
	if d != (game.ResumeDecision{RobotID: "p1", ReturnValue: game.BoolValue{Bool: false}}) {
		t.Error("move result was not delivered")
	}
}

func TestAtomicMovement(t *testing.T) {
	t.Parallel()
	forward := game.ForwardAction{}
	tests := []struct {
		name    string
		grid    [][]game.TerrainValue
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
			want: map[game.RobotID]board.RobotState{
				"a": state(1, 0, board.East), "b": state(1, 1, board.South),
				"c": state(0, 1, board.West), "d": state(0, 0, board.North),
			},
			moved: map[game.RobotID]bool{"a": true, "b": true, "c": true, "d": true},
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
			name: "goal",
			grid: [][]game.TerrainValue{{
				game.LandTerrain{Kind: game.PlainLand{}},
				game.LandTerrain{Kind: game.GoalLand{}},
			}},
			robots: map[game.RobotID]board.RobotState{"a": state(0, 0, board.East)},
			ids:    []game.RobotID{"a"},
			want:   map[game.RobotID]board.RobotState{"a": state(1, 0, board.East)},
			moved:  map[game.RobotID]bool{"a": true},
		},
		{
			name:   "rock",
			grid:   [][]game.TerrainValue{{game.LandTerrain{Kind: game.PlainLand{}}, game.RockTerrain{}}},
			robots: map[game.RobotID]board.RobotState{"a": state(0, 0, board.East)},
			ids:    []game.RobotID{"a"},
		},
		{
			name:   "sea",
			grid:   [][]game.TerrainValue{{game.LandTerrain{Kind: game.PlainLand{}}, game.SeaTerrain{}}},
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
				if d != (game.StartDecision{RobotID: id}) {
					t.Errorf("unexpected request: %+v", d)
				}
				var a game.Action = forward
				if custom, ok := tt.actions[id]; ok {
					a = custom
				}
				d = next(t, b, game.ActionReply{RobotID: id, Action: a})
			}
			if d != (game.AdvanceDecision{Tick: 1}) {
				t.Error("missing advance proposal")
			}
			d = next(t, b, nil)
			update, committed := d.(game.UpdateDecision)
			if !committed || b.Snapshot().Tick != 1 {
				t.Error("missing tick commit")
			}
			want := tt.want
			if want == nil {
				want = tt.robots
			}
			if !reflect.DeepEqual(b.Snapshot().Robots, want) {
				t.Errorf("got %+v, want %+v", b.Snapshot().Robots, want)
			}
			for _, result := range update.Update.Results {
				if _, rotating := result.Action.(game.RotateAction); rotating {
					if result.Value != (game.UnitValue{}) {
						t.Error("rotation must return unit")
					}
				} else if result.Value != (game.BoolValue{Bool: tt.moved[result.RobotID]}) {
					t.Errorf("unexpected movement result: %+v", result)
				}
			}
		})
	}
}

func TestWalkableTerrainKinds(t *testing.T) {
	t.Parallel()
	for _, kind := range []game.LandKind{
		game.PlainLand{}, game.GoalLand{}, game.CheckpointLand{Order: 2}, game.LuggageLand{},
	} {
		t.Run(fmt.Sprintf("%T", kind), func(t *testing.T) {
			t.Parallel()
			if _, isTerrain := any(kind).(game.TerrainValue); isTerrain {
				t.Errorf("%T must be nested inside LandTerrain, not used as terrain directly", kind)
			}
			terrain := game.LandTerrain{Kind: kind}
			b := newBoard(t, board.Config{
				Terrain: [][]game.TerrainValue{{terrain, terrain}},
				Robots:  map[game.RobotID]board.RobotState{"a": state(0, 0, board.East)},
			}, []game.RobotID{"a"})
			next(t, b, nil)
			next(t, b, game.ActionReply{RobotID: "a", Action: game.ForwardAction{}})
			d := next(t, b, nil)
			update, ok := d.(game.UpdateDecision)
			if !ok || len(update.Update.Results) != 1 {
				t.Errorf("unexpected commit: %+v", d)
				return
			}
			if update.Update.Results[0].Value != (game.BoolValue{Bool: true}) {
				t.Errorf("move onto %T failed: %+v", terrain, update.Update.Results[0])
			}
			snapshot := b.Snapshot()
			if snapshot.Robots["a"].Position != (board.Position{X: 1, Y: 0}) {
				t.Errorf("robot did not move onto %T: %+v", terrain, snapshot.Robots["a"])
			}
			if snapshot.Terrain[0][0] != terrain || snapshot.Terrain[0][1] != terrain {
				t.Errorf("movement changed terrain: %+v", snapshot.Terrain)
			}
		})
	}
}

func TestValidationAndCopies(t *testing.T) {
	t.Parallel()
	valid := board.Config{Terrain: land(2, 1), Robots: map[game.RobotID]board.RobotState{"a": state(0, 0, board.East)}}
	valid.Terrain[0][0] = game.LandTerrain{Kind: game.GoalLand{}}
	b, err := board.New(valid)
	if err != nil {
		t.Fatal(err)
	}
	initial := b.Snapshot()
	valid.Terrain[0][0] = game.SeaTerrain{}
	valid.Robots["a"] = state(1, 0, board.West)
	copy := b.Snapshot()
	copy.Terrain[0][0] = game.RockTerrain{}
	delete(copy.Robots, "a")
	if !reflect.DeepEqual(b.Snapshot(), initial) {
		t.Error("constructor or snapshot aliases caller data")
	}
	invalid := []board.Config{
		{}, {Terrain: land(2, 1)},
		{Terrain: [][]game.TerrainValue{{game.LandTerrain{Kind: game.PlainLand{}}}, {}}, Robots: initial.Robots},
		{Terrain: [][]game.TerrainValue{{nil}}, Robots: initial.Robots},
		{Terrain: [][]game.TerrainValue{{(*game.LandTerrain)(nil)}}, Robots: initial.Robots},
		{Terrain: [][]game.TerrainValue{{game.LandTerrain{}}}, Robots: initial.Robots},
		{Terrain: [][]game.TerrainValue{{game.LandTerrain{Kind: (*game.PlainLand)(nil)}}}, Robots: initial.Robots},
		{Terrain: [][]game.TerrainValue{{game.LandTerrain{Kind: &game.PlainLand{}}}}, Robots: initial.Robots},
		{Terrain: [][]game.TerrainValue{{game.LandTerrain{Kind: &game.GoalLand{}}}}, Robots: initial.Robots},
		{Terrain: [][]game.TerrainValue{{game.LandTerrain{Kind: &game.CheckpointLand{Order: 2}}}}, Robots: initial.Robots},
		{Terrain: [][]game.TerrainValue{{game.LandTerrain{Kind: &game.LuggageLand{}}}}, Robots: initial.Robots},
		{Terrain: [][]game.TerrainValue{{game.LandTerrain{Kind: (*game.GoalLand)(nil)}}}, Robots: initial.Robots},
		{Terrain: [][]game.TerrainValue{{&game.LandTerrain{Kind: game.GoalLand{}}}}, Robots: initial.Robots},
		{Terrain: [][]game.TerrainValue{{game.LandTerrain{Kind: (*game.CheckpointLand)(nil)}}}, Robots: initial.Robots},
		{Terrain: [][]game.TerrainValue{{&game.LandTerrain{Kind: game.CheckpointLand{Order: 2}}}}, Robots: initial.Robots},
		{Terrain: [][]game.TerrainValue{{game.LandTerrain{Kind: (*game.LuggageLand)(nil)}}}, Robots: initial.Robots},
		{Terrain: [][]game.TerrainValue{{&game.LandTerrain{Kind: game.LuggageLand{}}}}, Robots: initial.Robots},
		{Terrain: [][]game.TerrainValue{{game.SeaTerrain{}}}, Robots: initial.Robots},
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
	for _, reply := range []game.Reply{
		nil, (*game.ActionReply)(nil),
		game.DoneReply{RobotID: "other"},
		game.ActionReply{RobotID: "a"},
		game.ActionReply{RobotID: "a", Action: game.RotateAction{}},
		game.ActionReply{RobotID: "a", Action: game.RotateAction{Rotation: "sideways"}},
		game.ErrorReply{RobotID: "a"},
	} {
		if _, err := b.Next(t.Context(), reply); err == nil {
			t.Error("invalid reply accepted")
		}
		if !reflect.DeepEqual(initial, b.Snapshot()) {
			t.Error("invalid reply changed board")
		}
	}
	next(t, b, game.ActionReply{RobotID: "a", Action: game.ForwardAction{}})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := b.Next(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want cancellation", err)
	}
	if !reflect.DeepEqual(initial, b.Snapshot()) {
		t.Error("cancellation committed pending action")
	}
	if _, committed := next(t, b, nil).(game.UpdateDecision); !committed {
		t.Error("canceled commit could not be retried")
	}
	if b.Snapshot().Tick != 1 || b.Snapshot().Robots["a"].Position.X != 1 {
		t.Errorf("retry did not commit exactly once: %+v", b.Snapshot())
	}
}

func TestFinishedRobotRemainsAnObstacle(t *testing.T) {
	t.Parallel()
	b := newBoard(t, board.Config{Terrain: land(2, 1), Robots: map[game.RobotID]board.RobotState{"a": state(0, 0, board.East), "b": state(1, 0, board.East)}}, []game.RobotID{"a", "b"})
	next(t, b, nil)
	next(t, b, game.ActionReply{RobotID: "a", Action: game.ForwardAction{}})
	next(t, b, game.DoneReply{RobotID: "b"})
	d := next(t, b, nil)
	update, committed := d.(game.UpdateDecision)
	if !committed || len(update.Update.Results) != 1 || update.Update.Results[0].Value != (game.BoolValue{Bool: false}) {
		t.Error("finished robot did not block movement")
	}
	next(t, b, nil)
	if next(t, b, game.DoneReply{RobotID: "a"}) != (game.DoneDecision{}) {
		t.Error("board did not finish")
	}
}

func TestUnimplementedActionsAreNotSupported(t *testing.T) {
	t.Parallel()
	for _, action := range []game.Action{
		game.WaitAction{Integer: 7},
		game.PickUpAction{}, game.ReleaseAction{}, game.DrillAction{},
		game.DistanceAction{}, game.SearchFrontAction{}, game.SearchNowAction{},
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
			next(t, b, game.ActionReply{RobotID: "a", Action: game.ForwardAction{}})
			_, err := b.Next(t.Context(), game.ActionReply{RobotID: "b", Action: action})
			if !errors.Is(err, game.ErrNotSupported) {
				t.Errorf("got %v, want ErrNotSupported", err)
			}
			if !reflect.DeepEqual(initial, b.Snapshot()) {
				t.Error("unsupported action committed pending changes")
			}
		})
	}
}

func TestBoardLifecycle(t *testing.T) {
	t.Parallel()
	b, err := board.New(board.Config{
		Terrain: land(1, 1),
		Robots:  map[game.RobotID]board.RobotState{"a": state(0, 0, board.North)},
	})
	if err != nil {
		t.Fatal(err)
	}
	initial := b.Snapshot()
	if _, err := b.Next(t.Context(), nil); err == nil {
		t.Error("unstarted board advanced")
	}
	if err := b.Start(t.Context(), []game.RobotID{"a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Next(t.Context(), game.DoneReply{RobotID: "a"}); err == nil {
		t.Error("unsolicited reply was accepted")
	}
	if decision := next(t, b, nil); decision != (game.StartDecision{RobotID: "a"}) {
		t.Errorf("unexpected initial decision: %+v", decision)
	}
	if decision := next(t, b, game.ErrorReply{RobotID: "a", Err: errors.New("robot failed")}); decision != (game.DoneDecision{}) {
		t.Errorf("failed robot did not finish: %+v", decision)
	}
	if decision := next(t, b, nil); decision != (game.DoneDecision{}) {
		t.Errorf("completed board did not remain complete: %+v", decision)
	}
	if err := b.Start(t.Context(), []game.RobotID{"a"}); err == nil {
		t.Error("completed board restarted")
	}
	if _, err := b.Next(t.Context(), game.DoneReply{RobotID: "a"}); err == nil {
		t.Error("completed board accepted a reply")
	}
	if !reflect.DeepEqual(initial, b.Snapshot()) {
		t.Errorf("lifecycle checks changed the board: %+v", b.Snapshot())
	}
}
