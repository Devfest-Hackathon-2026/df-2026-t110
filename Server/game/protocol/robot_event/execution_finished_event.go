package robot_event

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
)

type ExecutionEndReason string

const (
	EndCompleted      ExecutionEndReason = "completed"
	EndInterrupted    ExecutionEndReason = "interrupted"
	EndRuntimeError   ExecutionEndReason = "runtime_error"
	EndDivisionByZero ExecutionEndReason = "division_by_zero"
	EndStepLimit      ExecutionEndReason = "step_limit"
	EndMemoryLimit    ExecutionEndReason = "memory_limit"
	EndCallDepthLimit ExecutionEndReason = "call_depth_limit"
)

// ExecutionFinishedEvent reports the end of a VM run after its cleanup events.
// Action suspensions are not execution endings. HintID identifies the last node.
type ExecutionFinishedEvent struct {
	HintID  protocol.HintID
	Reason  ExecutionEndReason
	Message string
}

func (ExecutionFinishedEvent) picolangEvent() {}
