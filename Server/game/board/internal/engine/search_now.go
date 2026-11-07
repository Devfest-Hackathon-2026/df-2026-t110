package engine

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func (b *Board) searchNow(id protocol.RobotID) values.LandTerrain {
	robot := b.state.Robots[id]
	position := robot.position
	// Robots always occupy land in the committed board state.
	land := b.state.Terrain[position.y][position.x].(LandTerrain)
	observation := values.LandTerrain{HasLuggage: land.hasLuggage}

	switch kind := land.kind.(type) {
	case PlainLand:
		observation.Kind = values.PlainLand{}
	case GoalLand:
		observation.Kind = values.GoalLand{}
	case CheckpointLand:
		observation.Kind = values.CheckpointLand{Order: kind.order}
	default:
		panic("invalid board land kind")
	}

	if robot.teamID != NoTeam && land.territoryOwner != NoTeam {
		observation.Territory = values.EnemyTerritory
		if robot.teamID == land.territoryOwner {
			observation.Territory = values.OwnTerritory
		}
	}

	return observation
}
