package domain

import "testing"

func TestRatingDelta(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                 string
		challenger, opponent int
		outcome              MatchOutcome
		want                 int
	}{
		{name: "win against much weaker", challenger: 1200, opponent: 1000, outcome: MatchWin, want: 5},
		{name: "win against equal", challenger: 1000, opponent: 1000, outcome: MatchWin, want: 30},
		{name: "win against much stronger", challenger: 1000, opponent: 1200, outcome: MatchWin, want: 120},
		{name: "win beyond limit is clamped", challenger: 1000, opponent: 1500, outcome: MatchWin, want: 120},
		{name: "win against slightly stronger truncates", challenger: 1000, opponent: 1100, outcome: MatchWin, want: 61},
		{name: "loss against much weaker", challenger: 1200, opponent: 1000, outcome: MatchLoss, want: -70},
		{name: "loss below limit is clamped", challenger: 1500, opponent: 1000, outcome: MatchLoss, want: -70},
		{name: "loss against equal", challenger: 1000, opponent: 1000, outcome: MatchLoss, want: -25},
		{name: "loss against much stronger", challenger: 1000, opponent: 1200, outcome: MatchLoss, want: -5},
		{name: "loss truncates toward zero, not down", challenger: 1000, opponent: 1050, outcome: MatchLoss, want: -18},
		{name: "exact loss is not lowered by float error", challenger: 1000, opponent: 1100, outcome: MatchLoss, want: -13},
		{name: "draw against equal", challenger: 1000, opponent: 1000, outcome: MatchDraw, want: 0},
		{name: "draw against much stronger", challenger: 1000, opponent: 1200, outcome: MatchDraw, want: 10},
		{name: "draw against much weaker", challenger: 1200, opponent: 1000, outcome: MatchDraw, want: -9},
		{name: "draw against slightly stronger floors", challenger: 1000, opponent: 1001, outcome: MatchDraw, want: 0},
		{name: "draw against slightly weaker floors", challenger: 1001, opponent: 1000, outcome: MatchDraw, want: 0},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := RatingDelta(tt.challenger, tt.opponent, tt.outcome); got != tt.want {
				t.Errorf("RatingDelta(%d, %d, %s)=%d, want %d", tt.challenger, tt.opponent, tt.outcome, got, tt.want)
			}
		})
	}
}

func TestOutcomeFromScores(t *testing.T) {
	t.Parallel()

	cases := []struct {
		challenger, opponent int64
		want                 MatchOutcome
	}{
		{challenger: 2, opponent: 1, want: MatchWin},
		{challenger: 1, opponent: 2, want: MatchLoss},
		{challenger: 1, opponent: 1, want: MatchDraw},
	}

	for _, tt := range cases {
		if got := OutcomeFromScores(tt.challenger, tt.opponent); got != tt.want {
			t.Errorf("OutcomeFromScores(%d, %d)=%s, want %s", tt.challenger, tt.opponent, got, tt.want)
		}
	}
}
