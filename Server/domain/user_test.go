package domain

import (
	"testing"
	"time"
)

func TestSessionExpired(t *testing.T) {
	t.Parallel()
	expiresAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{name: "before expiry", now: expiresAt.Add(-time.Nanosecond), want: false},
		{name: "at expiry", now: expiresAt, want: true},
		{name: "after expiry", now: expiresAt.Add(time.Nanosecond), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := (Session{ExpiresAt: expiresAt}).Expired(tt.now); got != tt.want {
				t.Errorf("Expired(%v) = %v, want %v", tt.now, got, tt.want)
			}
		})
	}
}
