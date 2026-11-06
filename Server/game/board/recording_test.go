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
				Terrain:  land(1, 1),
				Robots:   map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East)},
			})
			if err != nil {
				t.Fatal(err)
			}
			initialRecords, err := b.Start([]protocol.RobotID{"a"})
			if err != nil {
				t.Fatal(err)
			}
			wantInitial := []fieldevent.FieldEvent{{}}
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
			if output := call(robotmsg.ActionReply{RobotID: "a", Action: actions.DrillDurabilityAction{}}); len(output.Events) != 0 {
				t.Error("query emitted a field change")
			}
			rotate := robotmsg.ActionReply{RobotID: "a", Action: actions.RotateAction{Rotation: values.Right}}
			for tick := uint64(1); tick <= 3; tick++ {
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
				wantRecord := fieldevent.FieldEvent{Tick: tick}
				if output.Decision != (boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.UnitValue{}}) || len(output.Events) != 1 || output.Events[0] != wantRecord || b.Snapshot().Tick != tick {
					t.Errorf("commit = %+v, snapshot %+v", output, b.Snapshot())
				}
			}
			if !reflect.DeepEqual(initialRecords, wantInitial) {
				t.Error("initial records changed after progression")
			}
		})
	}
}
