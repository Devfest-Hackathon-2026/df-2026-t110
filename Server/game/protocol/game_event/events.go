// Package game_event defines history events for fields and robots.
package game_event

import (
	"encoding/json"
	"errors"

	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
)

// GameEvent is either a field event or an identified robot event.
type GameEvent interface {
	json.Marshaler
	gameEvent()
}

type FieldEvent struct{ Event fieldevent.FieldEvent }
type RobotEvent struct {
	InterpreterID protocol.RobotID
	Event         robotevent.PicolangEvent
}

func (FieldEvent) gameEvent() {}
func (RobotEvent) gameEvent() {}

func (r FieldEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Event)
}

func (r RobotEvent) MarshalJSON() ([]byte, error) {
	if r.InterpreterID == "" || !robotevent.ValidPicolangEvent(r.Event) {
		return nil, errors.New("interpreter ID and a debug event are required")
	}
	return json.Marshal(struct {
		Type          string                   `json:"type"`
		InterpreterID protocol.RobotID         `json:"interpreterId"`
		Event         robotevent.PicolangEvent `json:"event"`
	}{Type: "picolang", InterpreterID: r.InterpreterID, Event: r.Event})
}
