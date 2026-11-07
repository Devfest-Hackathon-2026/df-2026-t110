package engine

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func applyRotation(id protocol.RobotID, robot *RobotState, rotation values.Rotation) fieldevent.RotateResult {
	quarterTurns := 1
	if rotation == values.Left {
		quarterTurns = 3
	}

	robot.direction = [...]Direction{North, East, South, West}[(directionIndex(robot.direction)+quarterTurns)%4]

	return fieldevent.RotateResult{
		RobotID:   id,
		Rotation:  rotation,
		Direction: string(robot.direction),
	}
}
