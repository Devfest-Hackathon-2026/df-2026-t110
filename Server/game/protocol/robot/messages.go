// Package robot defines messages sent by robots through the hub.
package robot

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

type Step interface{ step() }
type ActionStep struct{ Action actions.Action }
type DoneStep struct{}

func (ActionStep) step() {}
func (DoneStep) step()   {}

type Reply interface{ reply() }
type ActionReply struct {
	RobotID protocol.RobotID
	Action  actions.Action
}
type DoneReply struct{ RobotID protocol.RobotID }
type ErrorReply struct {
	RobotID protocol.RobotID
	Err     error
}

func (ActionReply) reply() {}
func (DoneReply) reply()   {}
func (ErrorReply) reply()  {}

type ActionResult struct {
	RobotID protocol.RobotID
	Action  actions.Action
	Value   values.Value
}

// RobotOutput contains independent messages for the board and the recorder.
// Events returned with an error are still delivered.
type RobotOutput struct {
	Step   Step
	Events []robotevent.PicolangEvent
}
