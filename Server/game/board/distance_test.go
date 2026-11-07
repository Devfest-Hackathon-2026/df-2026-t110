package board_test

import (
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func TestDistanceFindsTheNearestTargetOnTheForwardRay(t *testing.T) {
	t.Parallel()

	terrain := land(8, 2)
	terrain[0][1] = board.RockTerrain{}
	terrain[0][2] = board.NewLandTerrain(board.PlainLand{}, true, board.TeamID2)
	terrain[0][6] = board.SeaTerrain{}
	terrain[0][7] = board.NewLandTerrain(board.GoalLand{}, false, board.NoTeam)
	tests := []struct {
		name   string
		target values.DistanceTarget
		team   board.TeamID
		want   int64
	}{
		{
			name:   "nearest robot includes unassigned robots",
			target: values.DistanceRobot,
			team:   board.TeamID1,
			want:   2,
		},
		{
			name:   "ally",
			target: values.DistanceAlly,
			team:   board.TeamID1,
			want:   5,
		},
		{
			name:   "enemy",
			target: values.DistanceEnemy,
			team:   board.TeamID1,
			want:   3,
		},
		{
			name:   "unassigned observer has no allies",
			target: values.DistanceAlly,
			team:   board.NoTeam,
			want:   -1,
		},
		{
			name:   "unassigned observer has no enemies",
			target: values.DistanceEnemy,
			team:   board.NoTeam,
			want:   -1,
		},
		{
			name:   "rock",
			target: values.DistanceRock,
			team:   board.NoTeam,
			want:   1,
		},
		{
			name:   "luggage behind a rock",
			target: values.DistanceLuggage,
			team:   board.NoTeam,
			want:   2,
		},
		{
			name:   "sea inside the board",
			target: values.DistanceSea,
			team:   board.NoTeam,
			want:   6,
		},
		{
			name:   "goal beyond sea and territory",
			target: values.DistanceGoal,
			team:   board.NoTeam,
			want:   7,
		},
		{
			name:   "territory of another team",
			target: values.DistanceTerritory,
			team:   board.TeamID1,
			want:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reply := robotmsg.ActionReply{RobotID: "a", Action: actions.DistanceAction{Target: tt.target}}
			want := boardmsg.BoardOutput{
				Decision: boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.IntegerValue{Integer: tt.want}},
			}
			config, err := board.NewConfig(
				terrain,
				map[protocol.RobotID]board.RobotState{
					"a":       state(0, 0, board.East).WithTeam(tt.team),
					"neutral": state(2, 0, board.West),
					"enemy":   state(3, 0, board.West).WithTeam(board.TeamID2),
					"ally":    state(5, 0, board.West).WithTeam(board.TeamID1),
					"offRay":  state(1, 1, board.West).WithTeam(board.TeamID2),
				},
				0,
				board.AllStoppedRules{},
			)
			if err != nil {
				t.Fatal(err)
			}
			b := newBoard(t, config, []protocol.RobotID{"a", "neutral", "enemy", "ally", "offRay"})
			before := b.Snapshot()
			next(t, b, nil)

			output, err := b.Next(reply)
			after := b.Snapshot()

			if err != nil || !reflect.DeepEqual(output, want) || !reflect.DeepEqual(after, before) {
				t.Errorf("distance = %+v, %v, want %+v; board = %+v", output, err, want, after)
			}
		})
	}
}

func TestDistanceUsesTurnStartStateAndExcludesCarriedLuggage(t *testing.T) {
	t.Parallel()

	carrier := loadedRobot(t, "a", 0, board.East)
	reply := func(target values.DistanceTarget) robotmsg.ActionReply {
		return robotmsg.ActionReply{RobotID: "b", Action: actions.DistanceAction{Target: target}}
	}
	want := []boardmsg.Decision{
		boardmsg.ResumeDecision{RobotID: "b", ReturnValue: values.IntegerValue{Integer: 2}},
		boardmsg.ResumeDecision{RobotID: "b", ReturnValue: values.IntegerValue{Integer: -1}},
		boardmsg.ResumeDecision{RobotID: "b", ReturnValue: values.IntegerValue{Integer: 3}},
	}
	config, err := board.NewConfig(
		land(3, 1),
		map[protocol.RobotID]board.RobotState{"a": carrier, "b": state(2, 0, board.West)},
		0,
		board.AllStoppedRules{},
	)
	if err != nil {
		t.Fatal(err)
	}
	b := newBoard(t, config, []protocol.RobotID{"a", "b"})
	before := b.Snapshot()
	next(t, b, nil)
	next(t, b, robotmsg.ActionReply{RobotID: "a", Action: actions.ForwardAction{}})

	invalid, err := b.Next(reply("invalid"))
	decisions := []boardmsg.Decision{
		next(t, b, reply(values.DistanceRobot)),
		next(t, b, reply(values.DistanceLuggage)),
		next(t, b, reply(values.DistanceSea)),
	}
	after := b.Snapshot()

	if err == nil || invalid.Decision != nil || len(invalid.Events) != 0 {
		t.Errorf("invalid target = %+v, %v", invalid, err)
	}
	if !reflect.DeepEqual(decisions, want) || !reflect.DeepEqual(after, before) {
		t.Errorf("distance decisions = %+v, want %+v; board = %+v", decisions, want, after)
	}
}
