package robot_event

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
)

type VariableDropEvent struct{ HintID protocol.HintID }

func (VariableDropEvent) picolangEvent() {}
