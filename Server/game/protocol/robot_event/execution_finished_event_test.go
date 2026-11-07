package robot_event_test

import (
	json "encoding/json/v2"
	"fmt"
	"testing"

	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
)

func TestExecutionFinishedHistoryEncoding(t *testing.T) {
	t.Parallel()
	for _, reason := range []robotevent.ExecutionEndReason{
		robotevent.EndCompleted, robotevent.EndInterrupted, robotevent.EndRuntimeError,
		robotevent.EndDivisionByZero, robotevent.EndStepLimit, robotevent.EndMemoryLimit, robotevent.EndCallDepthLimit,
	} {
		t.Run(string(reason), func(t *testing.T) {
			t.Parallel()
			end := robotevent.ExecutionFinishedEvent{HintID: hint, Reason: reason, Message: "execution result"}
			if !robotevent.ValidPicolangEvent(end) || robotevent.ValidPicolangEvent(&end) {
				t.Error("history validation must accept value events and reject pointer variants")
			}
			encoded, err := json.Marshal(gameevent.RobotEvent{InterpreterID: "robot-a", Event: end})
			want := fmt.Sprintf(`{"type":"picolang","interpreterId":"robot-a","event":{"kind":"execution_finished","hintId":%q,"reason":%q,"message":"execution result"}}`, hint, reason)
			if err != nil || string(encoded) != want {
				t.Errorf("history event = %s, %v; want %s", encoded, err, want)
			}
		})
	}
}

func TestInvalidExecutionFinishedEvents(t *testing.T) {
	t.Parallel()
	for name, end := range map[string]robotevent.ExecutionFinishedEvent{
		"missing hint":    {Reason: robotevent.EndCompleted, Message: "completed"},
		"invalid hint":    {HintID: "invalid", Reason: robotevent.EndCompleted, Message: "completed"},
		"missing reason":  {HintID: hint, Message: "completed"},
		"unknown reason":  {HintID: hint, Reason: "unknown", Message: "completed"},
		"missing message": {HintID: hint, Reason: robotevent.EndCompleted},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := json.Marshal(end); err == nil {
				t.Error("invalid execution ending was encoded")
			}
		})
	}
}
