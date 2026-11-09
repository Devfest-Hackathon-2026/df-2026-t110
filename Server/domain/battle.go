package domain

import "encoding/json/jsontext"

const BattleRobotCount = 3

// BattleTeam retains the robot order, including repeated programs.
type BattleTeam struct {
	User     User
	Programs []Program
}

// BattleLayout is a complete initial board and the rules used for one match.
type BattleLayout struct {
	ID, FieldID                   string
	Terrain                       [][]BattleCell
	Robots                        [2][BattleRobotCount]BattleRobot
	DrillLimit                    int64
	Turns                         uint64
	LuggagePoints, DiscardPenalty int64
}

type BattleCell struct {
	Terrain        string
	HasLuggage     bool
	TerritoryOwner int
}

type BattleRobot struct {
	X, Y      int
	Direction string
}

// Match identifies a completed simulation whose result is available via RunID.
type Match struct {
	ID               string
	RunID            string
	FieldID          string
	SelectedLayoutID string
	OpponentName     string
}

// MatchRecord is saved atomically with its result after the engine completes.
type MatchRecord struct {
	GoogleSubject    string
	OpponentSubject  string
	OpponentName     string
	FieldID          string
	SelectedLayoutID string
	Result           jsontext.Value
}
