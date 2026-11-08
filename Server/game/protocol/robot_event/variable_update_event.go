package robot_event

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
)

type VariableUpdateEvent struct {
	HintID protocol.HintID
	Diff   []ValueUpdate
}

func (VariableUpdateEvent) picolangEvent() {}
