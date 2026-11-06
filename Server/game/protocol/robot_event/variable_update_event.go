package robot_event

import (
	"encoding/json"
	"errors"

	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
)

type VariableUpdateEvent struct {
	HintID protocol.HintID
	Diff   []ValueUpdate
}

func (VariableUpdateEvent) picolangEvent() {}

func (r VariableUpdateEvent) MarshalJSON() ([]byte, error) {
	if err := validateHintID(r.HintID); err != nil {
		return nil, err
	}
	if len(r.Diff) == 0 {
		return nil, errors.New("variable update requires a nonempty diff")
	}
	diff := make([]json.RawMessage, len(r.Diff))
	for i, update := range r.Diff {
		encoded, err := marshalValueUpdate(update)
		if err != nil {
			return nil, err
		}
		diff[i] = encoded
	}
	return json.Marshal(struct {
		Kind   string            `json:"kind"`
		HintID protocol.HintID   `json:"hintId"`
		Diff   []json.RawMessage `json:"diff"`
	}{Kind: "variable_update", HintID: r.HintID, Diff: diff})
}
