package robot_event

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	memory "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/memory"
)

// VariableInitEvent pushes a binding for this definition without changing reference counts.
type VariableInitEvent struct {
	HintID  protocol.HintID
	ValueID memory.ID
	PicolangEvent
}
