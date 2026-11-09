package robot_event

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	memory "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/memory"
)

// VariableUpdateEvent changes the innermost binding without changing reference counts.
type VariableUpdateEvent struct {
	HintID  protocol.HintID
	ValueID memory.ID
	PicolangEvent
}
