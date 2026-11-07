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
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func land(width, height int) [][]board.Terrain {
	plain := board.NewLandTerrain(board.PlainLand{}, false, 0)
	grid := make([][]board.Terrain, height)
	for y := range grid {
		grid[y] = make([]board.Terrain, width)
		for x := range grid[y] {
			grid[y][x] = plain
		}
	}
	return grid
}

func state(x, y int, direction board.Direction) board.RobotState {
	return board.NewRobotState(board.NewPosition(x, y), direction, 3)
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
	plain := board.NewLandTerrain(board.PlainLand{}, false, 0)
	tests := []struct {
		name    string
		grid    [][]board.Terrain
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
			want: map[protocol.RobotID]board.RobotState{
				"a": state(1, 0, board.East), "b": state(1, 1, board.South),
				"c": state(0, 1, board.West), "d": state(0, 0, board.North),
			},
			moved: map[protocol.RobotID]bool{"a": true, "b": true, "c": true, "d": true},
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
			grid: [][]board.Terrain{{
				plain,
				board.NewLandTerrain(board.GoalLand{}, false, 0),
			}},
			robots: map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East)},
			ids:    []protocol.RobotID{"a"},
			want:   map[protocol.RobotID]board.RobotState{"a": state(1, 0, board.East)},
			moved:  map[protocol.RobotID]bool{"a": true},
		},
		{
			name:   "rock",
			grid:   [][]board.Terrain{{plain, board.RockTerrain{}}},
			robots: map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East)},
			ids:    []protocol.RobotID{"a"},
		},
		{
			name:   "sea",
			grid:   [][]board.Terrain{{plain, board.SeaTerrain{}}},
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

			wantRobots := tt.want
			if wantRobots == nil {
				wantRobots = tt.robots
			}

			turnReplies := []robotmsg.Reply{nil}
			var completionReplies []robotmsg.Reply
			var wantTurnOutputs []boardmsg.BoardOutput
			var wantResumes []boardmsg.BoardOutput
			var wantUpdate fieldevent.UpdateState

			for _, id := range tt.ids {
				var action actions.Action = forward
				if custom, ok := tt.actions[id]; ok {
					action = custom
				}

				turnReplies = append(turnReplies, robotmsg.ActionReply{RobotID: id, Action: action})
				completionReplies = append(completionReplies, robotmsg.DoneReply{RobotID: id})
				wantTurnOutputs = append(wantTurnOutputs, boardmsg.BoardOutput{
					Decision: boardmsg.StartDecision{RobotID: id},
				})

				var value values.Value = values.BoolValue{Bool: tt.moved[id]}
				if rotate, ok := action.(actions.RotateAction); ok {
					value = values.UnitValue{}
					wantUpdate.Actions = append(wantUpdate.Actions, fieldevent.RotateResult{
						RobotID:   id,
						Rotation:  rotate.Rotation,
						Direction: string(wantRobots[id].Direction()),
					})
				} else {
					move := fieldevent.MoveResult{
						Kind:    fieldevent.Forward,
						RobotID: id,
						Success: tt.moved[id],
					}
					if _, ok := action.(actions.BackAction); ok {
						move.Kind = fieldevent.Back
					}
					if move.Success {
						position := wantRobots[id].Position()
						move.To = &fieldevent.Position{X: position.X(), Y: position.Y()}
					}
					wantUpdate.Actions = append(wantUpdate.Actions, move)
				}

				wantResumes = append(wantResumes, boardmsg.BoardOutput{
					Decision: boardmsg.ResumeDecision{RobotID: id, ReturnValue: value},
				})
			}

			firstResume := wantResumes[0]
			firstResume.Events = []fieldevent.FieldEvent{{Tick: 1, State: wantUpdate}}
			wantTurnOutputs = append(wantTurnOutputs, firstResume)

			wantCompletionOutputs := append([]boardmsg.BoardOutput(nil), wantResumes[1:]...)
			wantCompletionOutputs = append(wantCompletionOutputs, boardmsg.BoardOutput{
				Decision: boardmsg.DoneDecision{},
				Events: []fieldevent.FieldEvent{{Tick: 1, State: fieldevent.UpdateState{
					RemovedRobotIDs: tt.ids,
				}}},
			})

			b := newBoard(t, board.Config{Terrain: tt.grid, Robots: tt.robots}, tt.ids)

			turnOutputs := make([]boardmsg.BoardOutput, len(turnReplies))
			for i, reply := range turnReplies {
				output, err := b.Next(reply)
				if err != nil {
					t.Fatal(err)
				}
				turnOutputs[i] = output
			}
			committed := b.Snapshot()

			completionOutputs := make([]boardmsg.BoardOutput, len(completionReplies))
			for i, reply := range completionReplies {
				output, err := b.Next(reply)
				if err != nil {
					t.Fatal(err)
				}
				completionOutputs[i] = output
			}
			finished := b.Snapshot()

			if !reflect.DeepEqual(turnOutputs, wantTurnOutputs) {
				t.Errorf("turn outputs = %+v, want %+v", turnOutputs, wantTurnOutputs)
			}
			if committed.Tick != 1 {
				t.Error("missing internal commit")
			}
			if !reflect.DeepEqual(committed.Robots, wantRobots) {
				t.Errorf("got %+v, want %+v", committed.Robots, wantRobots)
			}
			if !reflect.DeepEqual(completionOutputs, wantCompletionOutputs) {
				t.Errorf("completion outputs = %+v, want %+v", completionOutputs, wantCompletionOutputs)
			}
			if finished.Tick != 1 || len(finished.Robots) != 0 {
				t.Errorf("unexpected finished state: %+v", finished)
			}
		})
	}
}

func TestWalkableTerrainKinds(t *testing.T) {
	t.Parallel()
	for _, terrain := range []board.LandTerrain{
		board.NewLandTerrain(board.PlainLand{}, false, 0),
		board.NewLandTerrain(board.GoalLand{}, false, 0),
		board.NewLandTerrain(board.NewCheckpointLand(2), false, 0),
		board.NewLandTerrain(board.PlainLand{}, true, board.TeamID1),
	} {
		t.Run(fmt.Sprintf("%T(luggage=%t)", terrain.Kind(), terrain.HasLuggage()), func(t *testing.T) {
			t.Parallel()
			if _, isTerrain := any(terrain.Kind()).(board.Terrain); isTerrain {
				t.Errorf("%T must be nested inside LandTerrain, not used as terrain directly", terrain.Kind())
			}
			b := newBoard(t, board.Config{
				Terrain: [][]board.Terrain{{terrain, terrain}},
				Robots:  map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East)},
			}, []protocol.RobotID{"a"})
			next(t, b, nil)
			d := next(t, b, robotmsg.ActionReply{RobotID: "a", Action: actions.ForwardAction{}})
			if d != (boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.BoolValue{Bool: true}}) {
				t.Errorf("move onto %T failed: %+v", terrain, d)
			}
			snapshot := b.Snapshot()
			if snapshot.Robots["a"].Position() != board.NewPosition(1, 0) {
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
	goal := board.NewLandTerrain(board.GoalLand{}, false, 0)
	valid.Terrain[0][0] = goal
	b, err := board.New(valid)
	if err != nil {
		t.Fatal(err)
	}
	initial := b.Snapshot()
	valid.Terrain[0][0] = board.SeaTerrain{}
	valid.Robots["a"] = state(1, 0, board.West)
	copy := b.Snapshot()
	copy.Terrain[0][0] = board.RockTerrain{}
	delete(copy.Robots, "a")
	if !reflect.DeepEqual(b.Snapshot(), initial) {
		t.Error("constructor or snapshot aliases caller data")
	}
	plain := board.NewLandTerrain(board.PlainLand{}, false, 0)
	checkpoint := board.NewCheckpointLand(2)
	checkpointLand := board.NewLandTerrain(checkpoint, false, 0)
	luggage := board.NewLandTerrain(board.PlainLand{}, true, 0)
	invalid := []board.Config{
		{}, {Terrain: land(2, 1)},
		{Terrain: [][]board.Terrain{{plain}, {}}, Robots: initial.Robots},
		{Terrain: [][]board.Terrain{{nil}}, Robots: initial.Robots},
		{Terrain: [][]board.Terrain{{(*board.LandTerrain)(nil)}}, Robots: initial.Robots},
		{Terrain: [][]board.Terrain{{board.LandTerrain{}}}, Robots: initial.Robots},
		{Terrain: [][]board.Terrain{{board.NewLandTerrain((*board.PlainLand)(nil), false, 0)}}, Robots: initial.Robots},
		{Terrain: [][]board.Terrain{{board.NewLandTerrain(&board.PlainLand{}, false, 0)}}, Robots: initial.Robots},
		{Terrain: [][]board.Terrain{{board.NewLandTerrain(&board.GoalLand{}, false, 0)}}, Robots: initial.Robots},
		{Terrain: [][]board.Terrain{{board.NewLandTerrain(&checkpoint, false, 0)}}, Robots: initial.Robots},
		{Terrain: [][]board.Terrain{{board.NewLandTerrain((*board.GoalLand)(nil), false, 0)}}, Robots: initial.Robots},
		{Terrain: [][]board.Terrain{{&goal}}, Robots: initial.Robots},
		{Terrain: [][]board.Terrain{{board.NewLandTerrain((*board.CheckpointLand)(nil), false, 0)}}, Robots: initial.Robots},
		{Terrain: [][]board.Terrain{{&checkpointLand}}, Robots: initial.Robots},
		{Terrain: [][]board.Terrain{{&luggage}}, Robots: initial.Robots},
		{Terrain: [][]board.Terrain{{board.NewLandTerrain(board.GoalLand{}, false, board.TeamID1)}}, Robots: initial.Robots},
		{Terrain: [][]board.Terrain{{board.SeaTerrain{}}}, Robots: initial.Robots},
		{Terrain: land(1, 1), Robots: map[protocol.RobotID]board.RobotState{"a": state(1, 0, board.North)}},
		{Terrain: land(1, 1), Robots: map[protocol.RobotID]board.RobotState{"a": state(0, 0, "invalid")}},
		{Terrain: land(1, 1), Robots: map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.North).WithTeam(3)}},
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
	if b.Snapshot().Tick != 1 || b.Snapshot().Robots["a"].Position().X() != 1 {
		t.Errorf("valid reply did not commit exactly once: %+v", b.Snapshot())
	}
}

func TestFinishedRobotRemovedAtCommit(t *testing.T) {
	t.Parallel()
	b := newBoard(t, board.Config{Terrain: land(2, 1), Robots: map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East), "b": state(1, 0, board.East)}}, []protocol.RobotID{"a", "b"})
	initial := b.Snapshot()
	next(t, b, nil)
	next(t, b, robotmsg.ActionReply{RobotID: "a", Action: actions.ForwardAction{}})
	if d := next(t, b, robotmsg.DoneReply{RobotID: "b"}); d != (boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.BoolValue{Bool: true}}) {
		t.Errorf("move into the vacated cell failed: %+v", d)
	}
	if robots := b.Snapshot().Robots; len(robots) != 1 || robots["a"].Position() != initial.Robots["b"].Position() {
		t.Errorf("unexpected robots after commit: %+v", robots)
	}
	if next(t, b, robotmsg.DoneReply{RobotID: "a"}) != (boardmsg.DoneDecision{}) {
		t.Error("board did not finish")
	}
	if snapshot := b.Snapshot(); snapshot.Tick != 1 || len(snapshot.Robots) != 0 {
		t.Errorf("unexpected state after all robots stopped: %+v", snapshot)
	}
}

func TestUnimplementedActionsAreNotSupported(t *testing.T) {
	t.Parallel()
	for _, action := range []actions.Action{
		actions.WaitAction{Integer: 7},
		actions.PickUpAction{}, actions.ReleaseAction{}, actions.DrillAction{},
		actions.DistanceAction{Target: values.DistanceRock}, actions.SearchFrontAction{},
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

	terrain := land(1, 1)
	ids := []protocol.RobotID{"a"}
	failure := robotmsg.ErrorReply{RobotID: "a", Err: errors.New("robot failed")}
	wantStart := boardmsg.BoardOutput{Decision: boardmsg.StartDecision{RobotID: "a"}}
	wantCompletion := boardmsg.BoardOutput{
		Decision: boardmsg.DoneDecision{},
		Events: []fieldevent.FieldEvent{{State: fieldevent.UpdateState{
			RemovedRobotIDs: ids,
		}}},
	}
	wantRepeatedCompletion := boardmsg.BoardOutput{Decision: boardmsg.DoneDecision{}}
	wantSnapshot := board.Snapshot{
		Terrain: terrain,
		Robots:  map[protocol.RobotID]board.RobotState{},
	}

	b, err := board.New(board.Config{
		Terrain: terrain,
		Robots:  map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.North)},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, beforeStartErr := b.Next(nil)
	_, startErr := b.Start(ids)
	_, unsolicitedErr := b.Next(robotmsg.DoneReply{RobotID: "a"})
	startOutput, startNextErr := b.Next(nil)

	completionOutput, completionErr := b.Next(failure)
	repeatedCompletionOutput, repeatedCompletionErr := b.Next(nil)
	_, restartErr := b.Start(ids)
	_, completedReplyErr := b.Next(robotmsg.DoneReply{RobotID: "a"})
	finished := b.Snapshot()

	if beforeStartErr == nil {
		t.Error("unstarted board advanced")
	}
	if startErr != nil {
		t.Errorf("start failed: %v", startErr)
	}
	if unsolicitedErr == nil {
		t.Error("unsolicited reply was accepted")
	}
	if startNextErr != nil || !reflect.DeepEqual(startOutput, wantStart) {
		t.Errorf("initial output = %+v, %v", startOutput, startNextErr)
	}

	if completionErr != nil || !reflect.DeepEqual(completionOutput, wantCompletion) {
		t.Errorf("failed robot did not finish at tick zero: %+v, %v", completionOutput, completionErr)
	}
	if repeatedCompletionErr != nil || !reflect.DeepEqual(repeatedCompletionOutput, wantRepeatedCompletion) {
		t.Errorf("completed board did not remain complete: %+v, %v", repeatedCompletionOutput, repeatedCompletionErr)
	}
	if restartErr == nil {
		t.Error("completed board restarted")
	}
	if completedReplyErr == nil {
		t.Error("completed board accepted a reply")
	}

	if !reflect.DeepEqual(finished, wantSnapshot) {
		t.Errorf("expected only the stopped robot to be removed: %+v", finished)
	}
}
