package board

const EndAllLuggageDiscarded EndReason = "all_luggage_discarded"

type TeamResult struct {
	luggage  uint64
	discards uint64
	score    int64
}

func NewTeamResult(luggage, discards uint64, score int64) TeamResult {
	return TeamResult{luggage: luggage, discards: discards, score: score}
}

func (r TeamResult) Luggage() uint64  { return r.luggage }
func (r TeamResult) Discards() uint64 { return r.discards }
func (r TeamResult) Score() int64     { return r.score }

type BattleResult struct {
	tick   uint64
	reason EndReason
	team1  TeamResult
	team2  TeamResult
	winner int
}

func NewBattleResult(tick uint64, reason EndReason, team1, team2 TeamResult, winner int) BattleResult {
	return BattleResult{tick: tick, reason: reason, team1: team1, team2: team2, winner: winner}
}

func (r BattleResult) Tick() uint64         { return r.tick }
func (r BattleResult) EndReason() EndReason { return r.reason }
func (r BattleResult) Team1() TeamResult    { return r.team1 }
func (r BattleResult) Team2() TeamResult    { return r.team2 }

// Winner returns the winning team ID, or zero for a draw.
func (r BattleResult) Winner() int { return r.winner }
