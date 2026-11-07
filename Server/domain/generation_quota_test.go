package domain_test

import (
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

func TestGenerationQuotaRemaining(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name        string
		limit, used int
		want        int
	}{
		{"unused", 100, 0, 100},
		{"partly used", 100, 60, 40},
		{"exactly exhausted", 100, 100, 0},
		// The limit can be lowered after usage was recorded; the budget never goes negative.
		{"used beyond a lowered limit", 100, 130, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			q := domain.GenerationQuota{GoogleSubject: "sub", Limit: tt.limit, Used: tt.used}
			if got := q.Remaining(); got != tt.want {
				t.Errorf("Remaining() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGenerationQuotaCanReserve(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name        string
		limit, used int
		tokens      int
		want        bool
	}{
		{"within the remaining budget", 100, 60, 39, true},
		{"exactly the remaining budget", 100, 60, 40, true},
		{"one token over the remaining budget", 100, 60, 41, false},
		{"nothing remains", 100, 100, 1, false},
		{"used beyond a lowered limit", 100, 130, 1, false},
		{"zero tokens", 100, 0, 0, false},
		{"negative tokens", 100, 0, -1, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			q := domain.GenerationQuota{GoogleSubject: "sub", Limit: tt.limit, Used: tt.used}
			if got := q.CanReserve(tt.tokens); got != tt.want {
				t.Errorf("CanReserve(%d) = %v, want %v", tt.tokens, got, tt.want)
			}
		})
	}
}
