package domain_test

import (
	"errors"
	"math"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

func TestOpponentWeight(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                 string
		challenger, opponent int
		want                 float64
	}{
		{name: "equal rating", challenger: 1000, opponent: 1000, want: 1},
		{name: "one sigma stronger", challenger: 1000, opponent: 1150, want: math.Exp(-0.5)},
		{name: "one sigma weaker is symmetric", challenger: 1150, opponent: 1000, want: math.Exp(-0.5)},
		{name: "two sigma stronger", challenger: 1000, opponent: 1300, want: math.Exp(-2)},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := domain.OpponentWeight(tt.challenger, tt.opponent); math.Abs(got-tt.want) > 1e-12 {
				t.Errorf("OpponentWeight(%d, %d)=%v, want %v", tt.challenger, tt.opponent, got, tt.want)
			}
		})
	}
}

func TestSelectRatedOpponent(t *testing.T) {
	t.Parallel()

	me := domain.OpponentCandidate{GoogleSubject: "me", Rating: 1000}
	near := domain.OpponentCandidate{GoogleSubject: "near", Rating: 1000}
	sigma := domain.OpponentCandidate{GoogleSubject: "sigma", Rating: 1150}
	twoSigma := domain.OpponentCandidate{GoogleSubject: "two-sigma", Rating: 1300}
	far := domain.OpponentCandidate{GoogleSubject: "far", Rating: 7000}
	farther := domain.OpponentCandidate{GoogleSubject: "farther", Rating: 7100}
	// near and sigma weigh 1 and exp(-0.5), so near covers draws below 1/(1+exp(-0.5)) ≈ 0.6225.
	nearShare := 1 / (1 + math.Exp(-0.5))

	cases := []struct {
		name       string
		candidates []domain.OpponentCandidate
		draw       float64
		want       domain.OpponentCandidate
		err        error
	}{
		{name: "lowest draw picks the first candidate", candidates: []domain.OpponentCandidate{near, sigma}, draw: 0, want: near},
		{name: "draw just inside the nearer share", candidates: []domain.OpponentCandidate{near, sigma}, draw: nearShare - 1e-9, want: near},
		{name: "draw just past the nearer share", candidates: []domain.OpponentCandidate{near, sigma}, draw: nearShare + 1e-9, want: sigma},
		{name: "order does not change the shares", candidates: []domain.OpponentCandidate{sigma, near}, draw: 1 - nearShare + 1e-9, want: near},
		// Weights 1, exp(-0.5) and exp(-2) give sigma the draws in about [0.574, 0.922); a uniform pick would choose two-sigma.
		{name: "middle candidate owns its cumulative share", candidates: []domain.OpponentCandidate{near, sigma, twoSigma}, draw: 0.75, want: sigma},
		{name: "single candidate is always chosen", candidates: []domain.OpponentCandidate{sigma}, draw: 0.999, want: sigma},
		{name: "no hard limit on rating difference", candidates: []domain.OpponentCandidate{far}, draw: 0.5, want: far},
		// Both weights underflow to zero unless scaled, which would leave the choice to the last candidate.
		{name: "distant candidates keep relative weights", candidates: []domain.OpponentCandidate{far, farther}, draw: 0.999, want: far},
		{name: "challenger is excluded", candidates: []domain.OpponentCandidate{me, sigma}, draw: 0, want: sigma},
		{name: "only the challenger", candidates: []domain.OpponentCandidate{me}, draw: 0, err: domain.ErrNoOpponent},
		{name: "no candidates", candidates: nil, draw: 0, err: domain.ErrNoOpponent},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.SelectRatedOpponent(me.GoogleSubject, me.Rating, tt.candidates, func() float64 { return tt.draw })
			if !errors.Is(err, tt.err) {
				t.Errorf("err=%v, want %v", err, tt.err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
