package engine

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

type robotControl interface{ control() }

type unstartedControl struct{}
type readyControl struct{ value values.Value }
type pendingControl struct{ action actions.Action }
type waitingControl struct {
	remaining uint64

	robotControl
}
type finishedControl struct{}

func (unstartedControl) control() {}
func (readyControl) control()     {}
func (pendingControl) control()   {}
func (finishedControl) control()  {}

type phase interface{ phase() }

type unstartedBoard struct{}
type collectingBoard struct{}
type awaitingReplyBoard struct{ robotID protocol.RobotID }
type finishedBoard struct {
	decision boardmsg.Decision
	err      error
}

func (unstartedBoard) phase()     {}
func (collectingBoard) phase()    {}
func (awaitingReplyBoard) phase() {}
func (finishedBoard) phase()      {}
