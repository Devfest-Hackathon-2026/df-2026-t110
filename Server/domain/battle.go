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
}

// BattleLayout is a complete initial board and the rules used for one match.
type BattleLayout struct {
	ID             string                                           `json:"id"`
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
