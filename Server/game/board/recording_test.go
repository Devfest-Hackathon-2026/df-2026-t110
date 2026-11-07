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
			wantStart := boardmsg.StartDecision{RobotID: "a"}
			wantQuery := boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.IntegerValue{Integer: 3}}
			wantRotation := boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.UnitValue{}}
			queryReply := robotmsg.ActionReply{RobotID: "a", Action: actions.DrillDurabilityAction{}}
			rotateReply := robotmsg.ActionReply{RobotID: "a", Action: actions.RotateAction{Rotation: values.Right}}

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

			call := func(reply robotmsg.Reply) boardmsg.BoardOutput {
				t.Helper()
				output, err := b.Next(reply)
				if err != nil {
					t.Fatal(err)
				}
				return output
			}

			initialRecords, err := b.Start([]protocol.RobotID{"a"})
			if err != nil {
				t.Fatal(err)
			}
			start := call(nil)
			query := call(queryReply)

			if !reflect.DeepEqual(initialRecords, wantInitial) {
				t.Errorf("initial records = %+v", initialRecords)
			}
			if start.Decision != wantStart || len(start.Events) != 0 {
				t.Errorf("start request = %+v", start)
			}
			if query.Decision != wantQuery || len(query.Events) != 0 {
				t.Errorf("query output = %+v", query)
			}

			for i, direction := range []string{"south", "west", "north"} {
				tick := uint64(i + 1)
				wantRecords := []fieldevent.FieldEvent{{Tick: tick, State: fieldevent.UpdateState{
					Actions: []fieldevent.ActionResult{fieldevent.RotateResult{
						RobotID:   "a",
						Rotation:  values.Right,
						Direction: direction,
					}},
				}}}
				before := b.Snapshot()

				output, err := b.Next(rotateReply)
				after := b.Snapshot()

				if limit != 0 && tick > limit {
					if !errors.Is(err, board.ErrTickLimit) || len(output.Events) != 0 || !reflect.DeepEqual(before, after) {
						t.Errorf("limit did not prevent commit: %+v, %v, %+v", output, err, after)
					}
					break
				}
				if err != nil || output.Decision != wantRotation || !reflect.DeepEqual(output.Events, wantRecords) || after.Tick != tick {
					t.Errorf("commit = %+v, error %v, snapshot %+v", output, err, after)
				}
			}

			if !reflect.DeepEqual(initialRecords, wantInitial) {
				t.Error("initial records changed after progression")
			}
		})
	}
}
