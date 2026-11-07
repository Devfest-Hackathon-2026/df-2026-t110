// Package robot_event defines a single interpreter's debug events.
package robot_event

import "encoding/json"

// PicolangEvent reports an interpreter event without a clock or interpreter ID.
type PicolangEvent interface {
	json.Marshaler
	picolangEvent()
}

// ValidPicolangEvent rejects nil and pointer variants at recording boundaries.
func ValidPicolangEvent(message PicolangEvent) bool {
	switch message.(type) {
	case CurrentNodeEvent, VariableInitEvent, VariableUpdateEvent, VariableDropEvent, ExecutionFinishedEvent:
		return true
	default:
		return false
	}
}
