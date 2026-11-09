package domain

import "math"

// InitialRating is the rating of a user who has never challenged anyone.
const InitialRating = 1000

// ratingDiffLimit bounds the rating difference used in RatingDelta so that
// mismatched pairings neither reward nor penalize beyond the ±200 table.
const ratingDiffLimit = 200

// MatchOutcome is the challenger's result of a match.
type MatchOutcome string

const (
	MatchWin  MatchOutcome = "win"
	MatchLoss MatchOutcome = "loss"
	MatchDraw MatchOutcome = "draw"
)

// OutcomeFromScores judges the challenger's result: the higher final score wins and a tie is a draw.
func OutcomeFromScores(challenger, opponent int64) MatchOutcome {
	switch {
	case challenger > opponent:
		return MatchWin
	case challenger < opponent:
		return MatchLoss
	default:
		return MatchDraw
	}
}

// RatingDelta returns the change of the challenger's rating. Only the challenger is
// rated, so the result is intentionally not zero-sum.
func RatingDelta(challenger, opponent int, outcome MatchOutcome) int {
	d := float64(min(max(opponent-challenger, -ratingDiffLimit), ratingDiffLimit))
	switch outcome {
	case MatchWin:
		return truncate((450*math.Pow(3.6, d/ratingDiffLimit) - 60) / 13)
	case MatchLoss:
		return truncate(11 - 36*math.Pow(4.0/9, d/ratingDiffLimit))
	default:
		expected := 1 / (1 + math.Pow(10, d/400))
		switch {
		case d > 0:
			return int(math.Floor(40 * (0.5 - expected)))
		case d < 0:
			return -int(math.Floor(36 * (expected - 0.5)))
		default:
			return 0
		}
	}
}

// truncate rounds toward zero, absorbing floating-point error so that exact
// values such as 120 at the table boundaries are not truncated to 119.
func truncate(v float64) int {
	const epsilon = 1e-9
	return int(math.Trunc(v + math.Copysign(epsilon, v)))
}
