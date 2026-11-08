package robot_event

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

type VariableInitEvent struct {
	HintID protocol.HintID
	Value  values.RecordValue
}

func (VariableInitEvent) picolangEvent() {}
