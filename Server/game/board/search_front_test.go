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

func TestSearchFrontObservesTerrainAhead(t *testing.T) {
	t.Parallel()

	plain := board.NewLandTerrain(board.PlainLand{}, false, board.NoTeam)
	terrain := [][]board.Terrain{
		{plain, board.RockTerrain{}, plain},
		{board.NewLandTerrain(board.NewCheckpointLand(2), false, board.NoTeam), plain, board.NewLandTerrain(board.PlainLand{}, true, board.TeamID2)},
		{plain, board.SeaTerrain{}, plain},
	}
	center := board.NewPosition(1, 1)
	tests := []struct {
		name      string
		position  board.Position
		direction board.Direction
		team      board.TeamID
		want      values.TerrainValue
	}{
		{
			name:      "rock to the north",
			position:  center,
			direction: board.North,
			team:      board.NoTeam,
			want:      values.RockTerrain{},
		},
		{
			name:      "sea to the south",
			position:  center,
			direction: board.South,
			team:      board.NoTeam,
			want:      values.SeaTerrain{},
		},
		{
			name:      "checkpoint to the west",
			position:  center,
			direction: board.West,
			team:      board.NoTeam,
			want:      values.LandTerrain{Kind: values.CheckpointLand{Order: 2}},
		},
		{
			name:      "own territory to the east",
			position:  center,
			direction: board.East,
			team:      board.TeamID2,
			want: values.LandTerrain{
				Kind:       values.PlainLand{},
				HasLuggage: true,
				Territory:  values.OwnTerritory,
			},
		},
		{
			name:      "enemy territory to the east",
			position:  center,
			direction: board.East,
			team:      board.TeamID1,
			want: values.LandTerrain{
				Kind:       values.PlainLand{},
				HasLuggage: true,
				Territory:  values.EnemyTerritory,
			},
		},
		{
			name:      "off the board to the west",
			position:  board.NewPosition(0, 0),
			direction: board.West,
			team:      board.NoTeam,
			want:      values.SeaTerrain{},
		},
		{
			name:      "off the board to the south",
			position:  board.NewPosition(2, 2),
			direction: board.South,
			team:      board.NoTeam,
			want:      values.SeaTerrain{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reply := robotmsg.ActionReply{RobotID: "a", Action: actions.SearchFrontAction{}}
			want := boardmsg.BoardOutput{
				Decision: boardmsg.ResumeDecision{RobotID: "a", ReturnValue: tt.want},
			}
			b := newBoard(t, board.Config{
				Terrain: terrain,
				Robots: map[protocol.RobotID]board.RobotState{
					"a": board.NewRobotState(tt.position, tt.direction, 3).WithTeam(tt.team),
				},
			}, []protocol.RobotID{"a"})
			before := b.Snapshot()
			next(t, b, nil)

			output, err := b.Next(reply)
			after := b.Snapshot()

			if err != nil || !reflect.DeepEqual(output, want) {
				t.Errorf("search_front = %+v, %v, want %+v", output, err, want)
			}
			if !reflect.DeepEqual(after, before) {
				t.Errorf("unexpected board state after search_front: %+v", after)
			}
		})
	}
}
