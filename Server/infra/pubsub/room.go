// Package pubsub provides an in-process implementation of usecase.RoomEventBroker.
package pubsub

import (
	"context"
	"sync"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

// DefaultBufferSize is the number of events queued per subscriber.
const DefaultBufferSize = 16

var _ usecase.RoomEventBroker = (*RoomBroker)(nil)

// RoomBroker fans events out to subscribers in this process only; running
// several server instances would need a shared broker behind the same interface.
//
// Publish never blocks: a subscriber whose buffer is full misses the event, so
// one slow client cannot stall a room.
type RoomBroker struct {
	mu         sync.Mutex
	rooms      map[string]map[*subscription]struct{}
	bufferSize int
}

// NewRoomBroker returns a broker; bufferSize defaults to DefaultBufferSize when not positive.
func NewRoomBroker(bufferSize int) *RoomBroker {
	if bufferSize <= 0 {
		bufferSize = DefaultBufferSize
	}
	return &RoomBroker{rooms: map[string]map[*subscription]struct{}{}, bufferSize: bufferSize}
}

type subscription struct {
	broker *RoomBroker
	roomID string
	events chan domain.RoomEvent
	once   sync.Once
}

func (s *subscription) Events() <-chan domain.RoomEvent { return s.events }

func (s *subscription) Close() {
	s.once.Do(func() { s.broker.unsubscribe(s) })
}

func (b *RoomBroker) Subscribe(_ context.Context, roomID string) (usecase.RoomSubscription, error) {
	s := &subscription{broker: b, roomID: roomID, events: make(chan domain.RoomEvent, b.bufferSize)}
	b.mu.Lock()
	defer b.mu.Unlock()
	subs, ok := b.rooms[roomID]
	if !ok {
		subs = map[*subscription]struct{}{}
		b.rooms[roomID] = subs
	}
	subs[s] = struct{}{}
	return s, nil
}

// unsubscribe closes the channel under the lock so that Publish never sends on a closed channel.
func (b *RoomBroker) unsubscribe(s *subscription) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs := b.rooms[s.roomID]
	delete(subs, s)
	if len(subs) == 0 {
		delete(b.rooms, s.roomID)
	}
	close(s.events)
}

func (b *RoomBroker) Publish(_ context.Context, roomID string, event domain.RoomEvent) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for s := range b.rooms[roomID] {
		select {
		case s.events <- event:
		default:
		}
	}
	return nil
}

// SubscriberCount reports how many subscriptions the room currently has.
func (b *RoomBroker) SubscriberCount(roomID string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.rooms[roomID])
}
