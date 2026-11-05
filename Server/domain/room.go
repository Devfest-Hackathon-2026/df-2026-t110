package domain

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidEvent reports an event that cannot be delivered to subscribers.
var ErrInvalidEvent = errors.New("invalid event")

// RoomEvent is a named message for every subscriber of a room. Data must be
// JSON-encodable; its shape depends on Name and is not fixed yet.
type RoomEvent struct {
	Name string
	Data any
}

// Validate checks that the event can be framed as a Server-Sent Event, whose
// name travels on a single "event:" line.
func (e RoomEvent) Validate() error {
	if e.Name == "" || strings.ContainsAny(e.Name, "\r\n") {
		return fmt.Errorf("%w: name %q", ErrInvalidEvent, e.Name)
	}
	return nil
}
