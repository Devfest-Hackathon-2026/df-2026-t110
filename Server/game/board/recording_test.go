package board_test

import (
	"errors"
	"fmt"
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

func TestFieldMessagesAndBoardOwnedLimit(t *testing.T) {
	t.Parallel()
	for _, limit := range []uint64{0, 1} {
		t.Run(fmt.Sprintf("limit=%d", limit), func(t *testing.T) {
			t.Parallel()
			b, err := board.New(board.Config{
				MaxTicks: limit,
				Terrain: [][]board.Terrain{{
					board.NewLandTerrain(board.PlainLand{}, true, board.TeamID1),
					board.NewLandTerrain(board.GoalLand{}, false, 0),
					board.NewLandTerrain(board.NewCheckpointLand(0), false, 0),
					board.RockTerrain{}, board.SeaTerrain{},
				}},
				Robots: map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East)},
			})
			if err != nil {
				t.Fatal(err)
			}
			initialRecords, err := b.Start([]protocol.RobotID{"a"})
			if err != nil {
				t.Fatal(err)
			}
			checkpointOrder := int64(0)
			wantInitial := []fieldevent.FieldEvent{{State: fieldevent.InitialState{
				Terrain: [][]fieldevent.Cell{{
					{Type: "land", Kind: &fieldevent.LandKind{Type: "plain"}, HasLuggage: true, TerritoryOwner: int(board.TeamID1)},
					{Type: "land", Kind: &fieldevent.LandKind{Type: "goal"}},
					{Type: "land", Kind: &fieldevent.LandKind{Type: "checkpoint", Order: &checkpointOrder}},
					{Type: "rock"}, {Type: "sea"},
				}},
				Robots: []fieldevent.RobotState{{ID: "a", Direction: "east", DrillDurability: 3}},
			}}}
			if !reflect.DeepEqual(initialRecords, wantInitial) {
				t.Errorf("initial records = %+v", initialRecords)
			}
			call := func(reply robotmsg.Reply) boardmsg.BoardOutput {
				t.Helper()
				output, err := b.Next(reply)
				if err != nil {
					t.Fatal(err)
				}
				return output
			}
			if output := call(nil); output.Decision != (boardmsg.StartDecision{RobotID: "a"}) || len(output.Events) != 0 {
				t.Errorf("start request = %+v", output)
			}
			query := call(robotmsg.ActionReply{RobotID: "a", Action: actions.DrillDurabilityAction{}})
			if query.Decision != (boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.IntegerValue{Integer: 3}}) || len(query.Events) != 0 {
				t.Errorf("query output = %+v", query)
			}
			rotate := robotmsg.ActionReply{RobotID: "a", Action: actions.RotateAction{Rotation: values.Right}}
			for i, direction := range []string{"south", "west", "north"} {
				tick := uint64(i + 1)
				before := b.Snapshot()
				output, err := b.Next(rotate)
				if limit != 0 && tick > limit {
					if !errors.Is(err, board.ErrTickLimit) || len(output.Events) != 0 || !reflect.DeepEqual(before, b.Snapshot()) {
						t.Errorf("limit did not prevent commit: %+v, %v, %+v", output, err, b.Snapshot())
					}
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				wantRecords := []fieldevent.FieldEvent{{Tick: tick, State: fieldevent.UpdateState{
					Actions: []fieldevent.ActionResult{fieldevent.RotateResult{RobotID: "a", Rotation: values.Right, Direction: direction}},
				}}}
				if output.Decision != (boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.UnitValue{}}) || !reflect.DeepEqual(output.Events, wantRecords) || b.Snapshot().Tick != tick {
					t.Errorf("commit = %+v, snapshot %+v", output, b.Snapshot())
				}
			}
			if !reflect.DeepEqual(initialRecords, wantInitial) {
				t.Error("initial records changed after progression")
			}
		})
	}
}
