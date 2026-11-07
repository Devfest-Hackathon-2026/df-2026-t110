package engine

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
)

func printInteger(id protocol.RobotID, integer int64) fieldevent.PrintIntegerResult {
	return fieldevent.PrintIntegerResult{RobotID: id, Integer: integer}
}
