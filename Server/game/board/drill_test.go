package board_test

import (
	json "encoding/json/v2"
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

func TestDrillOpensTerrainForTheNextTurn(t *testing.T) {
	t.Parallel()

	plain := board.NewLandTerrain(board.PlainLand{}, false, board.NoTeam)
	rock := board.RockTerrain{}
	drill := robotmsg.ActionReply{RobotID: "a", Action: actions.DrillAction{}}
	forward := robotmsg.ActionReply{RobotID: "b", Action: actions.ForwardAction{}}
	want := boardmsg.BoardOutput{
		Decision: boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.BoolValue{Bool: true}},
		Events: []fieldevent.FieldEvent{{Tick: 1, State: fieldevent.UpdateState{
			Actions: []fieldevent.ActionResult{
				fieldevent.DrillResult{RobotID: "a", Success: true, Target: &fieldevent.Position{X: 1}, DrillDurability: 2},
				fieldevent.MoveResult{Kind: fieldevent.Forward, RobotID: "b", Success: false},
			},
		}}},
	}
	wantJSON := `[{"type":"field","tick":1,"state":{"kind":"update","actions":[{"kind":"drill","robotId":"a","success":true,"target":{"x":1,"y":0},"drillDurability":2},{"kind":"forward","robotId":"b","success":false}]}}]`
	config, err := board.NewConfig(
		[][]board.Terrain{{plain, rock, plain}},
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
	initial := b.Snapshot()
	next(t, b, nil)

	pending, pendingErr := b.Next(drill)
	beforeCommit := b.Snapshot()
	output, err := b.Next(forward)
	afterDrill := b.Snapshot()
	encoded, encodeErr := json.Marshal(output.Events)
	moveResult := next(t, b, robotmsg.DoneReply{RobotID: "a"})
	nextMove := next(t, b, forward)
	afterMove := b.Snapshot()

	if pendingErr != nil || pending.Decision != (boardmsg.StartDecision{RobotID: "b"}) || len(pending.Events) != 0 || !reflect.DeepEqual(beforeCommit, initial) {
		t.Errorf("drill committed early: %+v, %v", pending, pendingErr)
	}
	if err != nil || !reflect.DeepEqual(output, want) || encodeErr != nil || string(encoded) != wantJSON {
		t.Errorf("drill output = %+v, %v; JSON = %s, %v", output, err, encoded, encodeErr)
	}
	if afterDrill.Terrain[0][1] != plain || afterDrill.Robots["a"].DrillDurability() != 2 || afterDrill.Robots["b"].Position() != board.NewPosition(2, 0) {
		t.Errorf("unexpected board after drilling: %+v", afterDrill)
	}
	if moveResult != (boardmsg.ResumeDecision{RobotID: "b", ReturnValue: values.BoolValue{Bool: false}}) {
		t.Errorf("robot moved into a rock during the drilling turn: %+v", moveResult)
	}
	if nextMove != (boardmsg.ResumeDecision{RobotID: "b", ReturnValue: values.BoolValue{Bool: true}}) || afterMove.Robots["b"].Position() != board.NewPosition(1, 0) || afterMove.Tick != 2 {
		t.Errorf("robot could not enter the cleared tile: %+v, %+v", nextMove, afterMove)
	}
}

func TestDrillSucceedsForAllEligibleRobotsAtTheSameRock(t *testing.T) {
	t.Parallel()

	plain := board.NewLandTerrain(board.PlainLand{}, false, board.NoTeam)
	rock := board.RockTerrain{}
	target := &fieldevent.Position{X: 1}
	tests := []struct {
		name          string
		otherCapacity int64
		wantA         fieldevent.DrillResult
		wantB         fieldevent.DrillResult
	}{
		{
			name:          "both can drill",
			otherCapacity: 1,
			wantA:         fieldevent.DrillResult{RobotID: "a", Success: true, Target: target, DrillDurability: 0},
			wantB:         fieldevent.DrillResult{RobotID: "b", Success: true, Target: target, DrillDurability: 0},
		},
		{
			name:          "other drill is exhausted",
			otherCapacity: 0,
			wantA:         fieldevent.DrillResult{RobotID: "a", Success: true, Target: target, DrillDurability: 0},
			wantB:         fieldevent.DrillResult{RobotID: "b", DrillDurability: 0},
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
			wantSecond := boardmsg.ResumeDecision{RobotID: "b", ReturnValue: values.BoolValue{Bool: tt.wantB.Success}}
			config, err := board.NewConfig(
				[][]board.Terrain{{plain, rock, plain}},
				map[protocol.RobotID]board.RobotState{
					"a": board.NewRobotState(board.NewPosition(0, 0), board.East, 1),
					"b": board.NewRobotState(board.NewPosition(2, 0), board.West, tt.otherCapacity),
				},
				0,
				board.AllStoppedRules{},
			)
			if err != nil {
				t.Fatal(err)
			}
			b := newBoard(t, config, []protocol.RobotID{"a", "b"})
			next(t, b, nil)
			next(t, b, robotmsg.ActionReply{RobotID: "a", Action: actions.DrillAction{}})

			output, err := b.Next(robotmsg.ActionReply{RobotID: "b", Action: actions.DrillAction{}})
			snapshot := b.Snapshot()
			second := next(t, b, robotmsg.DoneReply{RobotID: "a"})

			if err != nil || !reflect.DeepEqual(output, want) {
				t.Errorf("drill output = %+v, %v, want %+v", output, err, want)
			}
			if second != wantSecond {
				t.Errorf("second drill = %+v, want %+v", second, wantSecond)
			}
			if snapshot.Terrain[0][1] != plain || snapshot.Robots["a"].DrillDurability() != tt.wantA.DrillDurability || snapshot.Robots["b"].DrillDurability() != tt.wantB.DrillDurability {
				t.Errorf("unexpected board after drilling: %+v", snapshot)
			}
		})
	}
}
