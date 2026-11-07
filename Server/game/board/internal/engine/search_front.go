package engine

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func (b *Board) searchFront(id protocol.RobotID) values.TerrainValue {
	robot := b.state.Robots[id]
	delta := directionDeltas[robot.direction]
	target := Position{x: robot.position.x + delta.x, y: robot.position.y + delta.y}

	switch cell := b.terrainAt(target).(type) {
	case LandTerrain:
		return observeLand(cell, robot.teamID)
	case RockTerrain:
		return values.RockTerrain{}
	case SeaTerrain:
		return values.SeaTerrain{}
	default:
		panic("invalid board terrain")
	}
}
