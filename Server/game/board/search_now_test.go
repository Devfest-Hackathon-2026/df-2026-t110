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

func TestSearchNowObservesCurrentLand(t *testing.T) {
	t.Parallel()

	territory := board.NewLandTerrain(board.PlainLand{}, true, board.TeamID2)
	tests := []struct {
		name string
		land board.LandTerrain
		team board.TeamID
		want values.LandTerrain
	}{
		{
			name: "own territory with luggage",
			land: territory,
			team: board.TeamID2,
			want: values.LandTerrain{
				Kind:       values.PlainLand{},
				HasLuggage: true,
				Territory:  values.OwnTerritory,
			},
		},
		{
			name: "enemy territory with luggage",
			land: territory,
			team: board.TeamID1,
			want: values.LandTerrain{
				Kind:       values.PlainLand{},
				HasLuggage: true,
				Territory:  values.EnemyTerritory,
			},
		},
		{
			name: "unassigned robot in territory",
			land: territory,
			team: board.NoTeam,
			want: values.LandTerrain{
				Kind:       values.PlainLand{},
				HasLuggage: true,
			},
		},
		{
			name: "plain land outside territory",
			land: board.NewLandTerrain(board.PlainLand{}, false, board.NoTeam),
			team: board.TeamID1,
			want: values.LandTerrain{Kind: values.PlainLand{}},
		},
		{
			name: "goal",
			land: board.NewLandTerrain(board.GoalLand{}, false, board.NoTeam),
			team: board.NoTeam,
			want: values.LandTerrain{Kind: values.GoalLand{}},
		},
		{
			name: "checkpoint",
			land: board.NewLandTerrain(board.NewCheckpointLand(2), true, board.NoTeam),
			team: board.NoTeam,
			want: values.LandTerrain{
				Kind:       values.CheckpointLand{Order: 2},
				HasLuggage: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reply := robotmsg.ActionReply{RobotID: "a", Action: actions.SearchNowAction{}}
			want := boardmsg.BoardOutput{
				Decision: boardmsg.ResumeDecision{RobotID: "a", ReturnValue: tt.want},
			}
			config, err := board.NewConfig(
				[][]board.Terrain{{board.SeaTerrain{}, tt.land}},
				map[protocol.RobotID]board.RobotState{"a": state(1, 0, board.West).WithTeam(tt.team)},
				0,
				board.AllStoppedRules{},
			)
			if err != nil {
				t.Fatal(err)
			}
			b := newBoard(t, config, []protocol.RobotID{"a"})
			before := b.Snapshot()
			next(t, b, nil)

			output, err := b.Next(reply)
			after := b.Snapshot()

			if err != nil || !reflect.DeepEqual(output, want) {
				t.Errorf("search_now = %+v, %v, want %+v", output, err, want)
			}
			if !reflect.DeepEqual(after, before) || after.Robots["a"].TeamID() != tt.team {
				t.Errorf("unexpected board state after search_now: %+v", after)
			}
		})
	}
}
