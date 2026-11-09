package domain

import "math"

// opponentRatingSigma is the provisional spread of the Gaussian matchmaking weight.
const opponentRatingSigma = 150

// OpponentCandidate is a user eligible to be challenged in a ranked battle.
type OpponentCandidate struct {
	GoogleSubject string
	Rating        int
}

// OpponentWeight returns exp(-d²/(2σ²)) where d is the rating difference, so
// that opponents with a closer rating are more likely to be chosen.
func OpponentWeight(challenger, opponent int) float64 {
	return math.Exp(-ratingDistance(challenger, opponent))
}

// SelectRatedOpponent chooses a candidate other than challenger with a
// probability proportional to OpponentWeight. No candidate is ruled out by its
// rating difference.
//
// draw returns a value in [0, 1) and is injected so the choice is deterministic in tests.
func SelectRatedOpponent(challenger string, rating int, candidates []OpponentCandidate, draw func() float64) (OpponentCandidate, error) {
	others := make([]OpponentCandidate, 0, len(candidates))
	nearest := math.Inf(1)
	for _, c := range candidates {
		if c.GoogleSubject == challenger {
			continue
		}
		others = append(others, c)
		nearest = min(nearest, ratingDistance(rating, c.Rating))
	}
	if len(others) == 0 {
		return OpponentCandidate{}, ErrNoOpponent
	}
	// Weights are scaled by the nearest candidate's so that they keep their ratios
	// even when every OpponentWeight would underflow to zero.
	weights := make([]float64, len(others))
	total := 0.0
	for i, c := range others {
		weights[i] = math.Exp(nearest - ratingDistance(rating, c.Rating))
		total += weights[i]
	}
	target := draw() * total
	for i, w := range weights {
		if target < w {
			return others[i], nil
		}
		target -= w
	}
	// Rounding can leave target at the total; the last candidate owns that edge.
	return others[len(others)-1], nil
}

// ratingDistance is the exponent d²/(2σ²) of the matchmaking weight.
func ratingDistance(challenger, opponent int) float64 {
	d := float64(opponent - challenger)
	return d * d / (2 * opponentRatingSigma * opponentRatingSigma)
}
