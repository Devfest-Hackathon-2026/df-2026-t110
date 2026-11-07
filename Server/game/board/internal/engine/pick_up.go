package engine

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func applyPickUp(id protocol.RobotID, robot *RobotState, terrain [][]Terrain, target Position, success bool) (values.BoolValue, fieldevent.PickUpResult) {
	event := fieldevent.PickUpResult{RobotID: id, Success: success}
	if success {
		land := terrain[target.y][target.x].(LandTerrain)
		land.hasLuggage = false
		terrain[target.y][target.x] = land
		robot.carryingLuggage = true
		event.Target = &fieldevent.Position{X: target.x, Y: target.y}
	}

	return values.BoolValue{Bool: success}, event
}
