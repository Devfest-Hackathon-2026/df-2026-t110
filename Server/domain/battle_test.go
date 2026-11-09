package domain_test

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

func TestMatchEntryRecord(t *testing.T) {
	t.Parallel()

	entry := domain.MatchEntry{
		ID: "match-id", GoogleSubject: "owner", OpponentSubject: "opponent", OpponentName: "Opponent",
		ChallengerRating: 1000, OpponentRating: 1200,
	}
	replay := jsontext.Value(`{"history":[]}`)

	cases := []struct {
		name        string
		scores      [2]int64
		wantOutcome domain.MatchOutcome
		wantDelta   int
	}{
		{name: "win against stronger", scores: [2]int64{2, 1}, wantOutcome: domain.MatchWin, wantDelta: 120},
		{name: "loss against stronger", scores: [2]int64{0, 1}, wantOutcome: domain.MatchLoss, wantDelta: -5},
		{name: "draw against stronger", scores: [2]int64{1, 1}, wantOutcome: domain.MatchDraw, wantDelta: 10},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := entry.Record(domain.MatchResult{ChallengerScore: tt.scores[0], OpponentScore: tt.scores[1], Replay: replay})

			if got.Outcome != tt.wantOutcome || got.RatingDelta != tt.wantDelta {
				t.Errorf("outcome=%s delta=%d, want %s delta=%d", got.Outcome, got.RatingDelta, tt.wantOutcome, tt.wantDelta)
			}
			if got.ID != "match-id" || got.GoogleSubject != "owner" || got.OpponentSubject != "opponent" || got.OpponentName != "Opponent" ||
				got.ChallengerRating != 1000 || got.OpponentRating != 1200 || string(got.Result) != string(replay) {
				t.Errorf("record=%+v does not carry the entry and replay", got)
			}
		})
	}
}
