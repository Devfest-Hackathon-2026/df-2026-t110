package engine

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func (b *Board) searchFront(id protocol.RobotID) values.TerrainValue {
	robot := b.state.Robots[id]
	delta := directionDeltas[robot.direction]
	x, y := robot.position.x+delta.x, robot.position.y+delta.y
	terrain := b.state.Terrain
	if y < 0 || y >= len(terrain) || x < 0 || x >= len(terrain[y]) {
		return values.SeaTerrain{}
	}

	switch cell := terrain[y][x].(type) {
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
