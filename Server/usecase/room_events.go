package usecase

import (
	"context"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

// RoomSubscription delivers the events of one room until Close is called.
// Events is closed after Close.
type RoomSubscription interface {
	Events() <-chan domain.RoomEvent
	Close()
}

// RoomEventBroker fans room events out to the room's current subscribers.
type RoomEventBroker interface {
	Subscribe(ctx context.Context, roomID string) (RoomSubscription, error)
	Publish(ctx context.Context, roomID string, event domain.RoomEvent) error
}

// RoomEvents lets clients follow what happens in a room.
type RoomEvents struct {
	broker RoomEventBroker
}

func NewRoomEvents(broker RoomEventBroker) *RoomEvents {
	return &RoomEvents{broker: broker}
}

// Join subscribes to the room. Rooms are not persisted yet, so any room ID is accepted.
func (r *RoomEvents) Join(ctx context.Context, roomID string) (RoomSubscription, error) {
	return r.broker.Subscribe(ctx, roomID)
}

// Publish sends event to the room's current subscribers.
func (r *RoomEvents) Publish(ctx context.Context, roomID string, event domain.RoomEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}
	return r.broker.Publish(ctx, roomID, event)
}
