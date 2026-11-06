package robot_event

import (
	"encoding/json"

	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
)

type CurrentNodeEvent struct{ HintID protocol.HintID }

func (CurrentNodeEvent) picolangEvent() {}

func (r CurrentNodeEvent) MarshalJSON() ([]byte, error) {
	if err := validateHintID(r.HintID); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Kind   string          `json:"kind"`
		HintID protocol.HintID `json:"hintId"`
	}{Kind: "current_node", HintID: r.HintID})
}
