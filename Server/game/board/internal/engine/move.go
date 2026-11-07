package engine

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func applyMove(id protocol.RobotID, robot *RobotState, kind fieldevent.MoveKind, target Position, moved bool) (values.BoolValue, fieldevent.MoveResult) {
	event := fieldevent.MoveResult{Kind: kind, RobotID: id, Success: moved}
	if moved {
		robot.position = target
		event.To = &fieldevent.Position{X: target.x, Y: target.y}
	}

	return values.BoolValue{Bool: moved}, event
}
