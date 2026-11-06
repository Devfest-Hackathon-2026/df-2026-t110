package robot_event

import (
	"encoding/json"

	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
)

type VariableDropEvent struct{ HintID protocol.HintID }

func (VariableDropEvent) picolangEvent() {}

func (r VariableDropEvent) MarshalJSON() ([]byte, error) {
	if err := validateHintID(r.HintID); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Kind   string          `json:"kind"`
		HintID protocol.HintID `json:"hintId"`
	}{Kind: "variable_drop", HintID: r.HintID})
}
