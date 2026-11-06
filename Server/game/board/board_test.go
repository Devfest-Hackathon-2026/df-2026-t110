package board_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game"
	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func land(width, height int) [][]values.TerrainValue {
	grid := make([][]values.TerrainValue, height)
	for y := range grid {
		grid[y] = make([]values.TerrainValue, width)
		for x := range grid[y] {
			grid[y][x] = values.LandTerrain{Kind: values.PlainLand{}}
		}
	}
	return grid
}

func state(x, y int, direction board.Direction) board.RobotState {
	return board.RobotState{Position: board.Position{X: x, Y: y}, Direction: direction, DrillDurability: 3}
}

func newBoard(t *testing.T, config board.Config, ids []protocol.RobotID) *board.Board {
	t.Helper()
	b, err := board.New(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Start(ids); err != nil {
		t.Fatal(err)
	}
	return b
}

func next(t *testing.T, b *board.Board, reply robotmsg.Reply) boardmsg.Decision {
	t.Helper()
	d, err := b.Next(reply)
	if err != nil {
		t.Fatal(err)
	}
	return d.Decision
}

func TestBarrierAndImmutableQueries(t *testing.T) {
	t.Parallel()
	b := newBoard(t, board.Config{Terrain: land(3, 1), Robots: map[protocol.RobotID]board.RobotState{
		"p1": state(0, 0, board.East), "p2": state(2, 0, board.West),
	}}, []protocol.RobotID{"p1", "p2"})
	initial := b.Snapshot()
	d := next(t, b, nil)
	if d != (boardmsg.StartDecision{RobotID: "p1"}) {
		t.Errorf("unexpected initial decision: %+v", d)
	}
	query := actions.DrillDurabilityAction{}
	for range 3 {
		d = next(t, b, robotmsg.ActionReply{RobotID: "p1", Action: query})
		if d != (boardmsg.ResumeDecision{RobotID: "p1", ReturnValue: values.IntegerValue{Integer: 3}}) {
			t.Errorf("unexpected query result: %+v", d)
		}
		if !reflect.DeepEqual(initial, b.Snapshot()) {
			t.Error("query changed the board")
		}
	}
	move := actions.ForwardAction{}
	d = next(t, b, robotmsg.ActionReply{RobotID: "p1", Action: move})
	if d != (boardmsg.StartDecision{RobotID: "p2"}) {
		t.Errorf("expected p2, got %+v", d)
	}
	d = next(t, b, robotmsg.ActionReply{RobotID: "p2", Action: move})
	if d != (boardmsg.ResumeDecision{RobotID: "p1", ReturnValue: values.BoolValue{Bool: false}}) || b.Snapshot().Tick != 1 {
		t.Errorf("board did not commit internally: %+v", d)
	}
	if !reflect.DeepEqual(initial.Robots, b.Snapshot().Robots) {
		t.Error("same destination must reject both movements")
	}
}

func TestAtomicMovement(t *testing.T) {
	t.Parallel()
	forward := actions.ForwardAction{}
	tests := []struct {
		name    string
		grid    [][]values.TerrainValue
		robots  map[protocol.RobotID]board.RobotState
		ids     []protocol.RobotID
		actions map[protocol.RobotID]actions.Action
		want    map[protocol.RobotID]board.RobotState
		moved   map[protocol.RobotID]bool
	}{
		{
			name:   "chain into empty cell",
			grid:   land(4, 1),
			robots: map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East), "b": state(1, 0, board.East)},
			ids:    []protocol.RobotID{"a", "b"},
			want:   map[protocol.RobotID]board.RobotState{"a": state(1, 0, board.East), "b": state(2, 0, board.East)},
			moved:  map[protocol.RobotID]bool{"a": true, "b": true},
		},
		{
			name:   "swap",
			grid:   land(2, 1),
			robots: map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East), "b": state(1, 0, board.West)},
			ids:    []protocol.RobotID{"b", "a"},
		},
		{
			name: "cycle",
			grid: land(2, 2),
			robots: map[protocol.RobotID]board.RobotState{
				"a": state(0, 0, board.East), "b": state(1, 0, board.South),
				"c": state(1, 1, board.West), "d": state(0, 1, board.North),
			},
			ids: []protocol.RobotID{"a", "b", "c", "d"},
		},
		{
			name:   "conflict propagates",
			grid:   land(4, 1),
			robots: map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East), "b": state(1, 0, board.East), "c": state(3, 0, board.West)},
			ids:    []protocol.RobotID{"a", "b", "c"},
		},
		{
			name:   "boundary propagates",
			grid:   land(2, 1),
			robots: map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East), "b": state(1, 0, board.East)},
			ids:    []protocol.RobotID{"a", "b"},
		},
		{
			name: "goal",
			grid: [][]values.TerrainValue{{
				values.LandTerrain{Kind: values.PlainLand{}},
				values.LandTerrain{Kind: values.GoalLand{}},
			}},
			robots: map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East)},
			ids:    []protocol.RobotID{"a"},
			want:   map[protocol.RobotID]board.RobotState{"a": state(1, 0, board.East)},
			moved:  map[protocol.RobotID]bool{"a": true},
		},
		{
			name:   "rock",
			grid:   [][]values.TerrainValue{{values.LandTerrain{Kind: values.PlainLand{}}, values.RockTerrain{}}},
			robots: map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East)},
			ids:    []protocol.RobotID{"a"},
		},
		{
			name:   "sea",
			grid:   [][]values.TerrainValue{{values.LandTerrain{Kind: values.PlainLand{}}, values.SeaTerrain{}}},
			robots: map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East)},
			ids:    []protocol.RobotID{"a"},
		},
		{
			name:    "back",
			grid:    land(2, 1),
			robots:  map[protocol.RobotID]board.RobotState{"a": state(1, 0, board.East)},
			ids:     []protocol.RobotID{"a"},
			actions: map[protocol.RobotID]actions.Action{"a": actions.BackAction{}},
			want:    map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East)},
			moved:   map[protocol.RobotID]bool{"a": true},
		},
		{
			name:    "rotate blocks follower",
			grid:    land(2, 1),
			robots:  map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East), "b": state(1, 0, board.North)},
			ids:     []protocol.RobotID{"a", "b"},
			actions: map[protocol.RobotID]actions.Action{"b": actions.RotateAction{Rotation: values.Left}},
			want:    map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East), "b": state(1, 0, board.West)},
		},
		{
			name:    "rotate right wraps",
			grid:    land(1, 1),
			robots:  map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.West)},
			ids:     []protocol.RobotID{"a"},
			actions: map[protocol.RobotID]actions.Action{"a": actions.RotateAction{Rotation: values.Right}},
			want:    map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.North)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := newBoard(t, board.Config{Terrain: tt.grid, Robots: tt.robots}, tt.ids)
			d := next(t, b, nil)
			for _, id := range tt.ids {
				if d != (boardmsg.StartDecision{RobotID: id}) {
					t.Errorf("unexpected request: %+v", d)
				}
				var a actions.Action = forward
				if custom, ok := tt.actions[id]; ok {
					a = custom
				}
				d = next(t, b, robotmsg.ActionReply{RobotID: id, Action: a})
			}
			if b.Snapshot().Tick != 1 {
				t.Error("missing internal commit")
			}
			want := tt.want
			if want == nil {
				want = tt.robots
			}
			if !reflect.DeepEqual(b.Snapshot().Robots, want) {
				t.Errorf("got %+v, want %+v", b.Snapshot().Robots, want)
			}
			for _, id := range tt.ids {
				var value values.Value = values.BoolValue{Bool: tt.moved[id]}
				if _, rotating := tt.actions[id].(actions.RotateAction); rotating {
					value = values.UnitValue{}
				}
				if d != (boardmsg.ResumeDecision{RobotID: id, ReturnValue: value}) {
					t.Errorf("unexpected action result: %+v", d)
				}
				d = next(t, b, robotmsg.DoneReply{RobotID: id})
			}
		})
	}
}

func TestWalkableTerrainKinds(t *testing.T) {
	t.Parallel()
	for _, kind := range []values.LandKind{
		values.PlainLand{}, values.GoalLand{}, values.CheckpointLand{Order: 2}, values.LuggageLand{},
	} {
		t.Run(fmt.Sprintf("%T", kind), func(t *testing.T) {
			t.Parallel()
			if _, isTerrain := any(kind).(values.TerrainValue); isTerrain {
				t.Errorf("%T must be nested inside LandTerrain, not used as terrain directly", kind)
			}
			terrain := values.LandTerrain{Kind: kind}
			b := newBoard(t, board.Config{
				Terrain: [][]values.TerrainValue{{terrain, terrain}},
				Robots:  map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East)},
			}, []protocol.RobotID{"a"})
			next(t, b, nil)
			d := next(t, b, robotmsg.ActionReply{RobotID: "a", Action: actions.ForwardAction{}})
			if d != (boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.BoolValue{Bool: true}}) {
				t.Errorf("move onto %T failed: %+v", terrain, d)
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
	valid := board.Config{Terrain: land(2, 1), Robots: map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East)}}
	valid.Terrain[0][0] = values.LandTerrain{Kind: values.GoalLand{}}
	b, err := board.New(valid)
	if err != nil {
		t.Fatal(err)
	}
	initial := b.Snapshot()
	valid.Terrain[0][0] = values.SeaTerrain{}
	valid.Robots["a"] = state(1, 0, board.West)
	copy := b.Snapshot()
	copy.Terrain[0][0] = values.RockTerrain{}
	delete(copy.Robots, "a")
	if !reflect.DeepEqual(b.Snapshot(), initial) {
		t.Error("constructor or snapshot aliases caller data")
	}
	invalid := []board.Config{
		{}, {Terrain: land(2, 1)},
		{Terrain: [][]values.TerrainValue{{values.LandTerrain{Kind: values.PlainLand{}}}, {}}, Robots: initial.Robots},
		{Terrain: [][]values.TerrainValue{{nil}}, Robots: initial.Robots},
		{Terrain: [][]values.TerrainValue{{(*values.LandTerrain)(nil)}}, Robots: initial.Robots},
		{Terrain: [][]values.TerrainValue{{values.LandTerrain{}}}, Robots: initial.Robots},
		{Terrain: [][]values.TerrainValue{{values.LandTerrain{Kind: (*values.PlainLand)(nil)}}}, Robots: initial.Robots},
		{Terrain: [][]values.TerrainValue{{values.LandTerrain{Kind: &values.PlainLand{}}}}, Robots: initial.Robots},
		{Terrain: [][]values.TerrainValue{{values.LandTerrain{Kind: &values.GoalLand{}}}}, Robots: initial.Robots},
		{Terrain: [][]values.TerrainValue{{values.LandTerrain{Kind: &values.CheckpointLand{Order: 2}}}}, Robots: initial.Robots},
		{Terrain: [][]values.TerrainValue{{values.LandTerrain{Kind: &values.LuggageLand{}}}}, Robots: initial.Robots},
		{Terrain: [][]values.TerrainValue{{values.LandTerrain{Kind: (*values.GoalLand)(nil)}}}, Robots: initial.Robots},
		{Terrain: [][]values.TerrainValue{{&values.LandTerrain{Kind: values.GoalLand{}}}}, Robots: initial.Robots},
		{Terrain: [][]values.TerrainValue{{values.LandTerrain{Kind: (*values.CheckpointLand)(nil)}}}, Robots: initial.Robots},
		{Terrain: [][]values.TerrainValue{{&values.LandTerrain{Kind: values.CheckpointLand{Order: 2}}}}, Robots: initial.Robots},
		{Terrain: [][]values.TerrainValue{{values.LandTerrain{Kind: (*values.LuggageLand)(nil)}}}, Robots: initial.Robots},
		{Terrain: [][]values.TerrainValue{{&values.LandTerrain{Kind: values.LuggageLand{}}}}, Robots: initial.Robots},
		{Terrain: [][]values.TerrainValue{{values.SeaTerrain{}}}, Robots: initial.Robots},
		{Terrain: land(1, 1), Robots: map[protocol.RobotID]board.RobotState{"a": state(1, 0, board.North)}},
		{Terrain: land(1, 1), Robots: map[protocol.RobotID]board.RobotState{"a": state(0, 0, "invalid")}},
		{Terrain: land(1, 1), Robots: map[protocol.RobotID]board.RobotState{"": state(0, 0, board.North)}},
		{Terrain: land(1, 1), Robots: map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.North), "b": state(0, 0, board.South)}},
	}
	for i, config := range invalid {
		if _, err := board.New(config); err == nil {
			t.Errorf("invalid config %d accepted", i)
		}
	}
	if _, err := b.Start([]protocol.RobotID{"missing"}); err == nil {
		t.Error("unknown robot accepted")
	}
	if _, err := b.Start([]protocol.RobotID{"a"}); err != nil {
		t.Error(err)
	}
	if _, err := b.Start([]protocol.RobotID{"a"}); err == nil {
		t.Error("second start accepted")
	}
}

func TestRejectedRepliesDoNotCommit(t *testing.T) {
	t.Parallel()
	b := newBoard(t, board.Config{Terrain: land(2, 1), Robots: map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East)}}, []protocol.RobotID{"a"})
	initial := b.Snapshot()
	next(t, b, nil)
	for _, reply := range []robotmsg.Reply{
		nil, (*robotmsg.ActionReply)(nil),
		robotmsg.DoneReply{RobotID: "other"},
		robotmsg.ActionReply{RobotID: "a"},
		robotmsg.ActionReply{RobotID: "a", Action: actions.RotateAction{}},
		robotmsg.ActionReply{RobotID: "a", Action: actions.RotateAction{Rotation: "sideways"}},
		robotmsg.ErrorReply{RobotID: "a"},
	} {
		if _, err := b.Next(reply); err == nil {
			t.Error("invalid reply accepted")
		}
		if !reflect.DeepEqual(initial, b.Snapshot()) {
			t.Error("invalid reply changed board")
		}
	}
	if d := next(t, b, robotmsg.ActionReply{RobotID: "a", Action: actions.ForwardAction{}}); d != (boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.BoolValue{Bool: true}}) {
		t.Error("valid reply failed after rejected replies")
	}
	if b.Snapshot().Tick != 1 || b.Snapshot().Robots["a"].Position.X != 1 {
		t.Errorf("valid reply did not commit exactly once: %+v", b.Snapshot())
	}
}

func TestFinishedRobotRemainsAnObstacle(t *testing.T) {
	t.Parallel()
	b := newBoard(t, board.Config{Terrain: land(2, 1), Robots: map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East), "b": state(1, 0, board.East)}}, []protocol.RobotID{"a", "b"})
	next(t, b, nil)
	next(t, b, robotmsg.ActionReply{RobotID: "a", Action: actions.ForwardAction{}})
	d := next(t, b, robotmsg.DoneReply{RobotID: "b"})
	if d != (boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.BoolValue{Bool: false}}) {
		t.Error("finished robot did not block movement")
	}
	if next(t, b, robotmsg.DoneReply{RobotID: "a"}) != (boardmsg.DoneDecision{}) {
		t.Error("board did not finish")
	}
}

func TestUnimplementedActionsAreNotSupported(t *testing.T) {
	t.Parallel()
	for _, action := range []actions.Action{
		actions.WaitAction{Integer: 7},
		actions.PickUpAction{}, actions.ReleaseAction{}, actions.DrillAction{},
		actions.DistanceAction{}, actions.SearchFrontAction{}, actions.SearchNowAction{},
		actions.PrintIntegerAction{Integer: -42}, actions.PrintBoolAction{Bool: true},
	} {
		t.Run(fmt.Sprintf("%T", action), func(t *testing.T) {
			t.Parallel()
			b := newBoard(t, board.Config{Terrain: land(3, 1), Robots: map[protocol.RobotID]board.RobotState{
				"a": state(0, 0, board.East), "b": state(2, 0, board.West),
			}}, []protocol.RobotID{"a", "b"})
			initial := b.Snapshot()
			next(t, b, nil)
			// A valid action is already pending when the other robot sends an
			// unimplemented action; neither action may mutate the board.
			next(t, b, robotmsg.ActionReply{RobotID: "a", Action: actions.ForwardAction{}})
			_, err := b.Next(robotmsg.ActionReply{RobotID: "b", Action: action})
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
		Robots:  map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.North)},
	})
	if err != nil {
		t.Fatal(err)
	}
	initial := b.Snapshot()
	if _, err := b.Next(nil); err == nil {
		t.Error("unstarted board advanced")
	}
	if _, err := b.Start([]protocol.RobotID{"a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Next(robotmsg.DoneReply{RobotID: "a"}); err == nil {
		t.Error("unsolicited reply was accepted")
	}
	if decision := next(t, b, nil); decision != (boardmsg.StartDecision{RobotID: "a"}) {
		t.Errorf("unexpected initial decision: %+v", decision)
	}
	if decision := next(t, b, robotmsg.ErrorReply{RobotID: "a", Err: errors.New("robot failed")}); decision != (boardmsg.DoneDecision{}) {
		t.Errorf("failed robot did not finish: %+v", decision)
	}
	if decision := next(t, b, nil); decision != (boardmsg.DoneDecision{}) {
		t.Errorf("completed board did not remain complete: %+v", decision)
	}
	if _, err := b.Start([]protocol.RobotID{"a"}); err == nil {
		t.Error("completed board restarted")
	}
	if _, err := b.Next(robotmsg.DoneReply{RobotID: "a"}); err == nil {
		t.Error("completed board accepted a reply")
	}
	if !reflect.DeepEqual(initial, b.Snapshot()) {
		t.Errorf("lifecycle checks changed the board: %+v", b.Snapshot())
	}
}
