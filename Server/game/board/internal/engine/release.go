package engine

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

type releaseTarget struct {
	position  Position
	discarded bool
}

func (b *Board) releaseDestination(robot RobotState, position Position) (releaseTarget, bool) {
	if !robot.carryingLuggage {
		return releaseTarget{}, false
	}

	switch cell := b.terrainAt(position).(type) {
	case LandTerrain:
		return releaseTarget{position: position}, !cell.hasLuggage
	case SeaTerrain:
		return releaseTarget{discarded: true}, true
	default:
		return releaseTarget{}, false
	}
}

func applyRelease(id protocol.RobotID, robot *RobotState, terrain [][]Terrain, target releaseTarget, success bool) (values.BoolValue, fieldevent.ReleaseResult) {
	event := fieldevent.ReleaseResult{RobotID: id, Success: success}
	if !success {
		return values.BoolValue{Bool: false}, event
	}

	robot.carryingLuggage = false
	event.Discarded = target.discarded
	if !target.discarded {
		p := target.position
		land := terrain[p.y][p.x].(LandTerrain)
		land.hasLuggage = true
		terrain[p.y][p.x] = land
		event.Target = &fieldevent.Position{X: p.x, Y: p.y}
	}
	return values.BoolValue{Bool: true}, event
}
