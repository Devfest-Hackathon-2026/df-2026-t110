package field_event

import (
	"encoding/json"
)

// FieldState is a placeholder until the field history schema is defined.
type FieldState struct{}

type FieldEvent struct {
	Tick  uint64
	State FieldState
}

func (r FieldEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type  string     `json:"type"`
		Tick  uint64     `json:"tick"`
		State FieldState `json:"state"`
	}{Type: "field", Tick: r.Tick, State: r.State})
}
