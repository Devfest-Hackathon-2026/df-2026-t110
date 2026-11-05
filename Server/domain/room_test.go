package domain

import (
	"errors"
	"testing"
)

func TestRoomEventValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		eventName string
		wantErr   error
	}{
		{name: "plain name", eventName: "match-started"},
		{name: "empty name", eventName: "", wantErr: ErrInvalidEvent},
		{name: "line feed", eventName: "a\nb", wantErr: ErrInvalidEvent},
		{name: "carriage return", eventName: "a\rb", wantErr: ErrInvalidEvent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := (RoomEvent{Name: tt.eventName}).Validate(); !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
