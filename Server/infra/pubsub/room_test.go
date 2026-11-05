package pubsub

import (
	"context"
	"sync"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

func subscribe(t *testing.T, b *RoomBroker, roomID string) usecase.RoomSubscription {
	t.Helper()
	s, err := b.Subscribe(context.Background(), roomID)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestPublishReachesOnlySubscribersOfTheRoom(t *testing.T) {
	t.Parallel()
	b := NewRoomBroker(1)
	first, second, other := subscribe(t, b, "room-1"), subscribe(t, b, "room-1"), subscribe(t, b, "room-2")
	event := domain.RoomEvent{Name: "match-started", Data: map[string]string{"matchId": "m"}}

	if err := b.Publish(context.Background(), "room-1", event); err != nil {
		t.Fatal(err)
	}

	for name, s := range map[string]usecase.RoomSubscription{"first": first, "second": second} {
		select {
		case got := <-s.Events():
			if got.Name != event.Name {
				t.Errorf("%s subscriber got %q, want %q", name, got.Name, event.Name)
			}
		default:
			t.Errorf("%s subscriber got no event", name)
		}
	}
	if len(other.Events()) != 0 {
		t.Error("subscriber of another room got the event")
	}
}

func TestPublishDropsEventsForFullSubscriberWithoutBlocking(t *testing.T) {
	t.Parallel()
	b := NewRoomBroker(1)
	s := subscribe(t, b, "room")

	for _, name := range []string{"first", "second"} {
		if err := b.Publish(context.Background(), "room", domain.RoomEvent{Name: name}); err != nil {
			t.Fatal(err)
		}
	}

	if got := (<-s.Events()).Name; got != "first" {
		t.Errorf("buffered event = %q, want first", got)
	}
	if len(s.Events()) != 0 {
		t.Error("event beyond the buffer was queued")
	}
}

func TestCloseUnsubscribes(t *testing.T) {
	t.Parallel()
	b := NewRoomBroker(1)
	s := subscribe(t, b, "room")
	kept := subscribe(t, b, "room")

	s.Close()
	s.Close() // idempotent

	if _, open := <-s.Events(); open {
		t.Error("events channel is open after Close")
	}
	if got := b.SubscriberCount("room"); got != 1 {
		t.Errorf("SubscriberCount() = %d, want 1", got)
	}
	kept.Close()
	if got := b.SubscriberCount("room"); got != 0 {
		t.Errorf("SubscriberCount() = %d, want 0", got)
	}
	if err := b.Publish(context.Background(), "room", domain.RoomEvent{Name: "after-close"}); err != nil {
		t.Errorf("Publish() to empty room error = %v", err)
	}
}

func TestConcurrentPublishAndClose(t *testing.T) {
	t.Parallel()
	b := NewRoomBroker(1)
	var wg sync.WaitGroup
	for range 32 {
		s := subscribe(t, b, "room")
		wg.Go(func() {
			if err := b.Publish(context.Background(), "room", domain.RoomEvent{Name: "e"}); err != nil {
				t.Errorf("Publish() error = %v", err)
			}
		})
		wg.Go(s.Close)
	}
	wg.Wait()

	if got := b.SubscriberCount("room"); got != 0 {
		t.Errorf("SubscriberCount() = %d, want 0", got)
	}
}
