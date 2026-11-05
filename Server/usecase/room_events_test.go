package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/pubsub"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

func TestPublishDeliversOnlyValidEvents(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		eventName string
		wantErr   error
	}{
		{name: "plain name", eventName: "match-started"},
		{name: "empty name", eventName: "", wantErr: domain.ErrInvalidEvent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			events := usecase.NewRoomEvents(pubsub.NewRoomBroker(1))
			sub, err := events.Join(context.Background(), "room-1")
			if err != nil {
				t.Fatal(err)
			}
			defer sub.Close()

			err = events.Publish(context.Background(), "room-1", domain.RoomEvent{Name: tt.eventName})

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Publish() error = %v, want %v", err, tt.wantErr)
			}
			if delivered := len(sub.Events()) > 0; delivered != (tt.wantErr == nil) {
				t.Errorf("delivered = %v, want %v", delivered, tt.wantErr == nil)
			}
		})
	}
}
