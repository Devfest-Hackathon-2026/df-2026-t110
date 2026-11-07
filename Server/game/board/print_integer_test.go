package board_test

import (
	json "encoding/json/v2"
	"errors"
	"fmt"
	"math"
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

func TestPrintCommitsWithOtherRobots(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name    string
		action  actions.Action
		result  fieldevent.ActionResult
		encoded string
	}
	var tests []testCase
	for _, integer := range []int64{0, -42, math.MinInt64, math.MaxInt64} {
		tests = append(tests, testCase{
			name:    fmt.Sprint(integer),
			action:  actions.PrintIntegerAction{Integer: integer},
			result:  fieldevent.PrintIntegerResult{RobotID: "a", Integer: integer},
			encoded: fmt.Sprintf(`{"kind":"print_integer","robotId":"a","integer":%d}`, integer),
		})
	}
	for _, value := range []bool{false, true} {
		tests = append(tests, testCase{
			name:    fmt.Sprint(value),
			action:  actions.PrintBoolAction{Bool: value},
			result:  fieldevent.PrintBoolResult{RobotID: "a", Bool: value},
			encoded: fmt.Sprintf(`{"kind":"print_bool","robotId":"a","bool":%t}`, value),
		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			firstReply := robotmsg.ActionReply{RobotID: "a", Action: tt.action}
			secondReply := robotmsg.ActionReply{RobotID: "b", Action: actions.PrintIntegerAction{Integer: 7}}
			wantFirst := boardmsg.BoardOutput{Decision: boardmsg.StartDecision{RobotID: "b"}}
			wantCommit := boardmsg.BoardOutput{
				Decision: boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.UnitValue{}},
				Events: []fieldevent.FieldEvent{{Tick: 1, State: fieldevent.UpdateState{
					Actions: []fieldevent.ActionResult{
						tt.result,
						fieldevent.PrintIntegerResult{RobotID: "b", Integer: 7},
					},
				}}},
			}
			wantSecondResume := boardmsg.BoardOutput{
				Decision: boardmsg.ResumeDecision{RobotID: "b", ReturnValue: values.UnitValue{}},
			}
			wantJSON := fmt.Sprintf(`[{"type":"field","tick":1,"state":{"kind":"update","actions":[%s,{"kind":"print_integer","robotId":"b","integer":7}]}}]`, tt.encoded)

			config, err := board.NewConfig(
				land(2, 1),
				map[protocol.RobotID]board.RobotState{
					"a": state(0, 0, board.East),
					"b": state(1, 0, board.West),
				},
				1,
				board.AllStoppedRules{},
			)
			if err != nil {
				t.Fatal(err)
			}
			b := newBoard(t, config, []protocol.RobotID{"a", "b"})
			initial := b.Snapshot()
			next(t, b, nil)

			first, firstErr := b.Next(firstReply)
			beforeCommit := b.Snapshot()
			committed, commitErr := b.Next(secondReply)
			afterCommit := b.Snapshot()
			encoded, encodeErr := json.Marshal(committed.Events)
			secondResume, resumeErr := b.Next(firstReply)
			limited, limitErr := b.Next(secondReply)
			afterLimit := b.Snapshot()

			if firstErr != nil || !reflect.DeepEqual(first, wantFirst) || !reflect.DeepEqual(beforeCommit, initial) {
				t.Errorf("print did not wait for the other robot: %+v, %v", first, firstErr)
			}
			if commitErr != nil || !reflect.DeepEqual(committed, wantCommit) {
				t.Errorf("committed output = %+v, %v, want %+v", committed, commitErr, wantCommit)
			}
			if afterCommit.Tick != 1 || !reflect.DeepEqual(afterCommit.Robots, initial.Robots) || !reflect.DeepEqual(afterCommit.Terrain, initial.Terrain) {
				t.Errorf("print changed the board beyond advancing one tick: %+v", afterCommit)
			}
			if encodeErr != nil || string(encoded) != wantJSON {
				t.Errorf("history = %s, %v, want %s", encoded, encodeErr, wantJSON)
			}
			if resumeErr != nil || !reflect.DeepEqual(secondResume, wantSecondResume) {
				t.Errorf("second robot did not resume with unit: %+v, %v", secondResume, resumeErr)
			}
			if !errors.Is(limitErr, board.ErrTickLimit) || len(limited.Events) != 0 || !reflect.DeepEqual(afterLimit, afterCommit) {
				t.Errorf("print exceeded the tick limit: %+v, %v", limited, limitErr)
			}
		})
	}
}
