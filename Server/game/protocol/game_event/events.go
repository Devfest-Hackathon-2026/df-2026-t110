// Package game_event defines history events for fields and robots.
package game_event

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
)

// GameEvent is either a field event or an identified robot event.
type GameEvent interface {
	gameEvent()
}

type FieldEvent struct{ Event fieldevent.FieldEvent }
type RobotEvent struct {
	InterpreterID protocol.RobotID
	Event         robotevent.PicolangEvent
}

func (FieldEvent) gameEvent() {}
func (RobotEvent) gameEvent() {}
