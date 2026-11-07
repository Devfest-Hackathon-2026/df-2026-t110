package robot_event

import (
	json "encoding/json/v2"
	"errors"

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

func (e ExecutionFinishedEvent) MarshalJSON() ([]byte, error) {
	if err := validateHintID(e.HintID); err != nil {
		return nil, err
	}
	switch e.Reason {
	case EndCompleted, EndInterrupted, EndRuntimeError, EndDivisionByZero, EndStepLimit, EndMemoryLimit, EndCallDepthLimit:
	default:
		return nil, errors.New("invalid execution end reason")
	}
	if e.Message == "" {
		return nil, errors.New("execution end message is required")
	}
	return json.Marshal(struct {
		Kind    string             `json:"kind"`
		HintID  protocol.HintID    `json:"hintId"`
		Reason  ExecutionEndReason `json:"reason"`
		Message string             `json:"message"`
	}{Kind: "execution_finished", HintID: e.HintID, Reason: e.Reason, Message: e.Message})
}
