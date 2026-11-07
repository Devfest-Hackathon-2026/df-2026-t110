package engine

import fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"

func (b *Board) initialFieldState() fieldevent.InitialState {
	terrain := make([][]fieldevent.Cell, len(b.state.Terrain))
	for y, row := range b.state.Terrain {
		terrain[y] = make([]fieldevent.Cell, len(row))
		for x, cell := range row {
			terrain[y][x] = recordCell(cell)
		}
	}

	robots := make([]fieldevent.RobotState, 0, len(b.order))
	for _, id := range b.order {
		robot := b.state.Robots[id]
		robots = append(robots, fieldevent.RobotState{
			ID:              id,
			Position:        fieldevent.Position{X: robot.position.x, Y: robot.position.y},
			Direction:       string(robot.direction),
			DrillDurability: robot.drillDurability,
		})
	}

	return fieldevent.InitialState{Terrain: terrain, Robots: robots}
}

func recordCell(cell Terrain) fieldevent.Cell {
	switch terrain := cell.(type) {
	case RockTerrain:
		return fieldevent.Cell{Type: "rock"}
	case SeaTerrain:
		return fieldevent.Cell{Type: "sea"}
	case LandTerrain:
		var kind fieldevent.LandKind
		switch land := terrain.kind.(type) {
		case PlainLand:
			kind = fieldevent.LandKind{Type: "plain"}
		case GoalLand:
			kind = fieldevent.LandKind{Type: "goal"}
		case CheckpointLand:
			kind = fieldevent.LandKind{Type: "checkpoint", Order: &land.order}
		default:
			panic("invalid board land kind")
		}
		return fieldevent.Cell{
			Type:           "land",
			Kind:           &kind,
			HasLuggage:     terrain.hasLuggage,
			TerritoryOwner: int(terrain.territoryOwner),
		}
	default:
		panic("invalid board terrain")
	}
}
