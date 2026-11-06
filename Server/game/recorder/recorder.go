// Package recorder stores game events in delivery order.
package recorder

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
)

// Recorder stores one run's events. Its zero value is ready to use.
// Calls must be sequential; recorded events must not be modified.
type Recorder struct {
	Events []gameevent.GameEvent
}

var _ game.Recorder = (*Recorder)(nil)

func (r *Recorder) Record(event gameevent.GameEvent) error {
	r.Events = append(r.Events, event)
	return nil
}
