package robot_event

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
)

type CurrentNodeEvent struct{ HintID protocol.HintID }

func (CurrentNodeEvent) picolangEvent() {}
