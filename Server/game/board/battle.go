package board

import (
	"fmt"

	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
)

// BattleConfig configures normal battle completion and scoring.
type BattleConfig struct {
	turns          uint64
	luggagePoints  int64
	discardPenalty int64
}

func NewBattleConfig(turns uint64, luggagePoints, discardPenalty int64) BattleConfig {
	return BattleConfig{turns: turns, luggagePoints: luggagePoints, discardPenalty: discardPenalty}
}

type battleState struct {
	config   BattleConfig
	discards [3]uint64
}

// CompletionRules creates independent scoring state for a single battle.
func (c BattleConfig) CompletionRules() (CompletionRules, error) {
	if c.turns == 0 || c.luggagePoints < 0 || c.discardPenalty < 0 {
		return nil, fmt.Errorf("battle requires positive turns and nonnegative scoring parameters")
	}
	return &battleState{config: c}, nil
}

func (b *battleState) TurnLimit() uint64 { return b.config.turns }

func (b *battleState) AfterTurn(state Snapshot, update fieldevent.UpdateState) bool {
	for _, action := range update.Actions {
		release, ok := action.(fieldevent.ReleaseResult)
		if ok && release.Discarded {
			team := state.Robots[release.RobotID].TeamID()
			b.discards[team]++
		}
	}
	return false
}

func (b *battleState) Result(state Snapshot, reason boardmsg.EndReason) (boardmsg.CompletionResult, error) {
	var luggage [3]uint64
	for _, row := range state.Terrain {
		for _, cell := range row {
			if land, ok := cell.(LandTerrain); ok && land.HasLuggage() && (land.TerritoryOwner() == TeamID1 || land.TerritoryOwner() == TeamID2) {
				luggage[land.TerritoryOwner()]++
			}
		}
	}

	var teams [3]boardmsg.TeamResult
	for _, team := range []TeamID{TeamID1, TeamID2} {
		discards := b.discards[team]
		score := int64(luggage[team])*b.config.luggagePoints - int64(discards)*b.config.discardPenalty
		teams[team] = boardmsg.NewTeamResult(luggage[team], discards, score)
	}

	team1, team2 := teams[TeamID1], teams[TeamID2]
	var winner TeamID
	switch {
	case team1.Score() > team2.Score():
		winner = TeamID1
	case team2.Score() > team1.Score():
		winner = TeamID2
	}
	return boardmsg.NewBattleResult(state.Tick, reason, team1, team2, int(winner)), nil
}
