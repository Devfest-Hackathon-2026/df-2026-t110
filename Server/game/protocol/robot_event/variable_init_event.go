package robot_event

import (
	"encoding/json"

	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

type VariableInitEvent struct {
	HintID protocol.HintID
	Value  values.RecordValue
}

func (VariableInitEvent) picolangEvent() {}

func (r VariableInitEvent) MarshalJSON() ([]byte, error) {
	if err := validateHintID(r.HintID); err != nil {
		return nil, err
	}
	value, err := marshalRecordValue(r.Value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Kind   string          `json:"kind"`
		HintID protocol.HintID `json:"hintId"`
		Value  json.RawMessage `json:"value"`
	}{Kind: "variable_init", HintID: r.HintID, Value: value})
}
