package domain

import (
	"encoding/json/jsontext"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
)

const BattleRobotCount = 3

// BattleTeam retains the robot order, including repeated programs.
type BattleTeam struct {
	User     User
	Programs []Program
	Rating   int
}

// BattleLayout is a complete initial board and the rules used for one match.
type BattleLayout struct {
	FieldID        string                                           `json:"fieldId"`
	Terrain        [][]schema.Terrain                               `json:"terrain"`
	Luggage        []schema.Position                                `json:"luggage"`
	Territories    [2][]schema.Position                             `json:"territories"`
	Robots         [2][BattleRobotCount]schema.FieldDefinitionRobot `json:"robots"`
	DrillLimit     int64                                            `json:"drillLimit"`
	Turns          uint64                                           `json:"turns"`
	LuggagePoints  int64                                            `json:"luggagePoints"`
	DiscardPenalty int64                                            `json:"discardPenalty"`
}

// Match identifies a completed simulation whose result is available via RunID.
type Match struct {
	ID           string
	RunID        string
	FieldID      string
	OpponentName string
}

// MatchResult is what the engine reports for a completed match.
type MatchResult struct {
	ChallengerScore int64
	OpponentScore   int64
	Replay          jsontext.Value
}

// MatchRecord is saved atomically with its result after the engine completes.
// RatingDelta is computed from the ratings read at matching time and is added to
// the challenger's rating current at saving time, so concurrent matches of one
// challenger never lose an update.
type MatchRecord struct {
	ID                 string
	ChallengerPrograms []ProgramRef
	OpponentPrograms   []ProgramRef
	GoogleSubject      string
	OpponentSubject    string
	OpponentName       string
	Result             jsontext.Value
	Outcome            MatchOutcome
	ChallengerRating   int
	OpponentRating     int
	RatingDelta        int
}

// MatchEntry is what matching fixes before the engine runs, including the ID
// the match is saved under.
type MatchEntry struct {
	ID               string
	GoogleSubject    string
	OpponentSubject  string
	OpponentName     string
	ChallengerRating int
	OpponentRating   int
}

// Record rates the completed match from the ratings read at matching time.
func (e MatchEntry) Record(result MatchResult) MatchRecord {
	outcome := OutcomeFromScores(result.ChallengerScore, result.OpponentScore)
	return MatchRecord{
		ID: e.ID, GoogleSubject: e.GoogleSubject, OpponentSubject: e.OpponentSubject, OpponentName: e.OpponentName,
		Result: result.Replay, Outcome: outcome,
		ChallengerRating: e.ChallengerRating, OpponentRating: e.OpponentRating,
		RatingDelta: RatingDelta(e.ChallengerRating, e.OpponentRating, outcome),
	}
}

// RankingEntry is a challenger's standing. Users sharing a rating share a rank.
type RankingEntry struct {
	Rank   int
	Name   string
	Rating int
	Wins   int
	Losses int
	Draws  int
}
