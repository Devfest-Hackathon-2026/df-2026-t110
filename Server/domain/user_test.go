package domain_test

import (
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

func TestIsValidUsername(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		valid bool
	}{
		{"a", true}, {"AZaz09_-", true}, {strings.Repeat("a", 32), true},
		{"", false}, {strings.Repeat("a", 33), false}, {"a b", false},
		{"a.b", false}, {"日本語", false}, {"a\n", false},
		{"self", false}, {"std", false}, {"Self", true}, {"std_user", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := domain.IsValidUsername(tt.name); got != tt.valid {
				t.Errorf("IsValidUsername(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}
