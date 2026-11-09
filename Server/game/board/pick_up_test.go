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

func TestPickUpCarriesAtMostOneItem(t *testing.T) {
	t.Parallel()

	plain := board.NewLandTerrain(board.PlainLand{}, false, board.NoTeam)
	territory := board.NewLandTerrain(board.PlainLand{}, false, board.TeamID2)
	territoryWithLuggage := board.NewLandTerrain(board.PlainLand{}, true, board.TeamID2)
	luggage := board.NewLandTerrain(board.PlainLand{}, true, board.NoTeam)
	pick := robotmsg.ActionReply{RobotID: "a", Action: actions.PickUpAction{}}
	wantPick := boardmsg.BoardOutput{
		Decision: boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.BoolValue{Bool: true}},
		Events: []fieldevent.FieldEvent{{Tick: 1, State: fieldevent.UpdateState{
			Actions: []fieldevent.ActionResult{fieldevent.PickUpResult{RobotID: "a", Success: true, Target: &fieldevent.Position{X: 1}}},
		}}},
	}
	wantFailure := boardmsg.BoardOutput{
		Decision: boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.BoolValue{Bool: false}},
		Events: []fieldevent.FieldEvent{{Tick: 3, State: fieldevent.UpdateState{
			Actions: []fieldevent.ActionResult{fieldevent.PickUpResult{RobotID: "a"}},
		}}},
	}
	wantTerrain := [][]board.Terrain{{plain, territory, luggage}}
	config, err := board.NewConfig(
		[][]board.Terrain{{plain, territoryWithLuggage, luggage}},
		map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East)},
		0,
		board.AllStoppedRules{},
	)
	if err != nil {
		t.Fatal(err)
	}
	b := newBoard(t, config, []protocol.RobotID{"a"})
	next(t, b, nil)

	picked, pickErr := b.Next(pick)
	first := b.Snapshot()
	next(t, b, robotmsg.ActionReply{RobotID: "a", Action: actions.ForwardAction{}})
	failed, failErr := b.Next(pick)
	last := b.Snapshot()
	next(t, b, robotmsg.DoneReply{RobotID: "a"})
	finished := b.Snapshot()

	if pickErr != nil || !reflect.DeepEqual(picked, wantPick) {
		t.Errorf("pickup output = %+v, %v, want %+v", picked, pickErr, wantPick)
	}
	if !first.Robots["a"].CarryingLuggage() || !reflect.DeepEqual(first.Terrain, wantTerrain) {
		t.Errorf("unexpected state after pickup: %+v", first)
	}
	if failErr != nil || !reflect.DeepEqual(failed, wantFailure) || !last.Robots["a"].CarryingLuggage() || !reflect.DeepEqual(last.Terrain, wantTerrain) {
		t.Errorf("second pickup = %+v, %v; board = %+v", failed, failErr, last)
	}
	if len(finished.Robots) != 0 || !reflect.DeepEqual(finished.Terrain, wantTerrain) {
		t.Errorf("finished robot's cargo should disappear: %+v", finished)
	}
}

func TestPickUpConflictLeavesLuggageInPlace(t *testing.T) {
	t.Parallel()

	plain := board.NewLandTerrain(board.PlainLand{}, false, board.NoTeam)
	luggage := board.NewLandTerrain(board.PlainLand{}, true, board.NoTeam)
	wantA := boardmsg.BoardOutput{
		Decision: boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.BoolValue{Bool: false}},
		Events: []fieldevent.FieldEvent{{Tick: 1, State: fieldevent.UpdateState{
			Actions: []fieldevent.ActionResult{
				fieldevent.PickUpResult{RobotID: "a"},
				fieldevent.PickUpResult{RobotID: "b"},
			},
		}}},
	}
	wantB := boardmsg.ResumeDecision{RobotID: "b", ReturnValue: values.BoolValue{Bool: false}}
	config, err := board.NewConfig(
		[][]board.Terrain{{plain, luggage, plain}},
		map[protocol.RobotID]board.RobotState{
			"a": state(0, 0, board.East),
			"b": state(2, 0, board.West),
		},
		0,
		board.AllStoppedRules{},
	)
	if err != nil {
		t.Fatal(err)
	}
	b := newBoard(t, config, []protocol.RobotID{"a", "b"})
	wantState := b.Snapshot()
	wantState.Tick = 1
	next(t, b, nil)

	next(t, b, robotmsg.ActionReply{RobotID: "a", Action: actions.PickUpAction{}})
	output, err := b.Next(robotmsg.ActionReply{RobotID: "b", Action: actions.PickUpAction{}})
	snapshot := b.Snapshot()
	second := next(t, b, robotmsg.DoneReply{RobotID: "a"})

	if err != nil || !reflect.DeepEqual(output, wantA) || second != wantB {
		t.Errorf("competing pickups = %+v, %v; second decision = %+v", output, err, second)
	}
	if !reflect.DeepEqual(snapshot, wantState) {
		t.Errorf("competing pickups changed the board: %+v", snapshot)
	}
}
