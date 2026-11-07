package domain_test

import (
	"errors"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

func TestSelectOpponent(t *testing.T) {
	t.Parallel()
	a := domain.Submission{GoogleSubject: "a", ProgramID: "pa"}
	b := domain.Submission{GoogleSubject: "b", ProgramID: "pb"}
	c := domain.Submission{GoogleSubject: "c", ProgramID: "pc"}
	first := func(int) int { return 0 }
	last := func(n int) int { return n - 1 }

	tests := []struct {
		name string
		subs []domain.Submission
		pick func(int) int
		want domain.Submission
		err  error
	}{
		{"excludes challenger from candidates", []domain.Submission{a, b, c}, first, b, nil},
		{"picks by index among the others", []domain.Submission{a, b, c}, last, c, nil},
		{"only the challenger submitted", []domain.Submission{a}, first, domain.Submission{}, domain.ErrNoOpponent},
		{"nobody submitted", nil, first, domain.Submission{}, domain.ErrNoOpponent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.SelectOpponent("a", tt.subs, tt.pick)
			if !errors.Is(err, tt.err) {
				t.Errorf("err = %v, want %v", err, tt.err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
