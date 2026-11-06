// Package game defines the contracts shared by robots, boards, and the hub.
package game

import (
	"errors"

	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

// ErrNotSupported identifies actions a concrete implementation does not support.
var ErrNotSupported = errors.New("not supported")

// Robot starts without input and resumes with its previous action's result.
// Both calls must terminate.
type Robot interface {
	Start() (robotmsg.RobotOutput, error)
	Next(values.Value) (robotmsg.RobotOutput, error)
}

// Board selects robots and owns game progression. StartDecision and
// ResumeDecision require that robot's Reply on the next call. The first Next
// after Start requires nil. Both calls must terminate. Instances are single-use.
type Board interface {
	Start([]protocol.RobotID) ([]fieldevent.FieldEvent, error)
	Next(robotmsg.Reply) (boardmsg.BoardOutput, error)
}

// Recorder stores independently produced game events in delivery order.
// Calls are sequential. Producers must not modify events after delivery.
// Recording failures stop the run.
type Recorder interface {
	Record(gameevent.GameEvent) error
}
