package board_test

import (
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func loadedRobot(t *testing.T, id protocol.RobotID, x int, direction board.Direction) board.RobotState {
	t.Helper()
	terrain := land(3, 1)
	terrain[0][1] = board.NewLandTerrain(board.PlainLand{}, true, board.NoTeam)
	config, err := board.NewConfig(
		terrain,
		map[protocol.RobotID]board.RobotState{id: state(x, 0, direction)},
		0,
		board.AllStoppedRules{},
	)
	if err != nil {
		t.Fatal(err)
	}
	b := newBoard(t, config, []protocol.RobotID{id})
	next(t, b, nil)
	next(t, b, robotmsg.ActionReply{RobotID: id, Action: actions.PickUpAction{}})
	robot := b.Snapshot().Robots[id]
	if !robot.CarryingLuggage() {
		t.Fatal("failed to prepare a robot carrying luggage")
	}
	return robot
}

func TestReleaseUsesTurnStartTerrain(t *testing.T) {
	t.Parallel()

	plain := board.NewLandTerrain(board.PlainLand{}, false, board.NoTeam)
	luggage := board.NewLandTerrain(board.PlainLand{}, true, board.NoTeam)
	sea := board.SeaTerrain{}
	target := &fieldevent.Position{X: 1}
	a := loadedRobot(t, "a", 0, board.East)
	b := loadedRobot(t, "b", 2, board.West)
	tests := []struct {
		name          string
		cell          board.Terrain
		secondRobot   board.RobotState
		secondAction  actions.Action
		wantA         fieldevent.ReleaseResult
		wantB         fieldevent.ActionResult
		wantBValue    values.Value
		wantCell      board.Terrain
		wantBCarrying bool
	}{
		{
			name:          "two carriers cannot place on the same cell",
			cell:          plain,
			secondRobot:   b,
			secondAction:  actions.ReleaseAction{},
			wantA:         fieldevent.ReleaseResult{RobotID: "a"},
			wantB:         fieldevent.ReleaseResult{RobotID: "b"},
			wantBValue:    values.BoolValue{Bool: false},
			wantCell:      plain,
			wantBCarrying: true,
		},
		{
			name:         "both carriers can discard into the same sea cell",
			cell:         sea,
			secondRobot:  b,
			secondAction: actions.ReleaseAction{},
			wantA:        fieldevent.ReleaseResult{RobotID: "a", Success: true, Discarded: true},
			wantB:        fieldevent.ReleaseResult{RobotID: "b", Success: true, Discarded: true},
			wantBValue:   values.BoolValue{Bool: true},
			wantCell:     sea,
		},
		{
			name:         "cannot pick up luggage placed on the same turn",
			cell:         plain,
			secondRobot:  state(2, 0, board.West),
			secondAction: actions.PickUpAction{},
			wantA:        fieldevent.ReleaseResult{RobotID: "a", Success: true, Target: target},
			wantB:        fieldevent.PickUpResult{RobotID: "b"},
			wantBValue:   values.BoolValue{Bool: false},
			wantCell:     luggage,
		},
		{
			name:          "cannot place where luggage is being picked up",
			cell:          luggage,
			secondRobot:   state(2, 0, board.West),
			secondAction:  actions.PickUpAction{},
			wantA:         fieldevent.ReleaseResult{RobotID: "a"},
			wantB:         fieldevent.PickUpResult{RobotID: "b", Success: true, Target: target},
			wantBValue:    values.BoolValue{Bool: true},
			wantCell:      plain,
			wantBCarrying: true,
		},
		{
			name:         "can place on land occupied by a robot",
			cell:         plain,
			secondRobot:  state(1, 0, board.West),
			secondAction: actions.RotateAction{Rotation: values.Right},
			wantA:        fieldevent.ReleaseResult{RobotID: "a", Success: true, Target: target},
			wantB:        fieldevent.RotateResult{RobotID: "b", Rotation: values.Right, Direction: "north"},
			wantBValue:   values.UnitValue{},
			wantCell:     luggage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			want := boardmsg.BoardOutput{
				Decision: boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.BoolValue{Bool: tt.wantA.Success}},
				Events: []fieldevent.FieldEvent{{Tick: 1, State: fieldevent.UpdateState{
					Actions: []fieldevent.ActionResult{tt.wantA, tt.wantB},
				}}},
			}
			wantSecond := boardmsg.ResumeDecision{RobotID: "b", ReturnValue: tt.wantBValue}
			config, err := board.NewConfig(
				[][]board.Terrain{{plain, tt.cell, plain}},
				map[protocol.RobotID]board.RobotState{"a": a, "b": tt.secondRobot},
				0,
				board.AllStoppedRules{},
			)
			if err != nil {
				t.Fatal(err)
			}
			board := newBoard(t, config, []protocol.RobotID{"a", "b"})
			next(t, board, nil)

			next(t, board, robotmsg.ActionReply{RobotID: "a", Action: actions.ReleaseAction{}})
			output, err := board.Next(robotmsg.ActionReply{RobotID: "b", Action: tt.secondAction})
			snapshot := board.Snapshot()
			second := next(t, board, robotmsg.DoneReply{RobotID: "a"})

			if err != nil || !reflect.DeepEqual(output, want) || second != wantSecond {
				t.Errorf("release output = %+v, %v; second decision = %+v", output, err, second)
			}
			if snapshot.Terrain[0][1] != tt.wantCell || snapshot.Robots["a"].CarryingLuggage() == tt.wantA.Success || snapshot.Robots["b"].CarryingLuggage() != tt.wantBCarrying {
				t.Errorf("unexpected board after release: %+v", snapshot)
			}
		})
	}
}

func TestReleaseOverTheEdgeDiscardsLuggage(t *testing.T) {
	t.Parallel()

	robot := loadedRobot(t, "a", 0, board.East)
	terrain := land(1, 1)
	reply := robotmsg.ActionReply{RobotID: "a", Action: actions.ReleaseAction{}}
	wantEvents := []fieldevent.FieldEvent{{Tick: 1, State: fieldevent.UpdateState{
		Actions: []fieldevent.ActionResult{fieldevent.ReleaseResult{RobotID: "a", Success: true, Discarded: true}},
	}}}
	config, err := board.NewConfig(
		terrain,
		map[protocol.RobotID]board.RobotState{"a": robot},
		0,
		board.AllStoppedRules{},
	)
	if err != nil {
		t.Fatal(err)
	}
	b := newBoard(t, config, []protocol.RobotID{"a"})
	next(t, b, nil)

	output, err := b.Next(reply)
	second := next(t, b, reply)
	snapshot := b.Snapshot()

	if err != nil || output.Decision != (boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.BoolValue{Bool: true}}) || !reflect.DeepEqual(output.Events, wantEvents) {
		t.Errorf("discard = %+v, %v; want events %+v", output, err, wantEvents)
	}
	if second != (boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.BoolValue{Bool: false}}) || snapshot.Tick != 2 || snapshot.Robots["a"].CarryingLuggage() || !reflect.DeepEqual(snapshot.Terrain, terrain) {
		t.Errorf("empty-handed release = %+v; board = %+v", second, snapshot)
	}
}
