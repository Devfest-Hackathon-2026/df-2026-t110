package engine

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
)

func printBool(id protocol.RobotID, value bool) fieldevent.PrintBoolResult {
	return fieldevent.PrintBoolResult{RobotID: id, Bool: value}
}
