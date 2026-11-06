// Package board defines messages sent by boards through the hub.
package board

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

// BoardOutput contains independent messages for robots and the recorder.
// Events returned with an error are still delivered.
type BoardOutput struct {
	Decision Decision
	Events   []fieldevent.FieldEvent
}

// Decision controls robot delivery independently of recording messages.
type Decision interface{ decision() }
type StartDecision struct{ RobotID protocol.RobotID }
type ResumeDecision struct {
	RobotID     protocol.RobotID
	ReturnValue values.Value
}
type DoneDecision struct{}

func (StartDecision) decision()  {}
func (ResumeDecision) decision() {}
func (DoneDecision) decision()   {}
