package board

type EndReason string

const (
	EndAllStopped   EndReason = "all_stopped"
	EndTurnLimit    EndReason = "turn_limit"
	EndConditionMet EndReason = "condition_met"
)

// CompletionResult is the mode-specific result of a normally completed game.
type CompletionResult interface {
	EndReason() EndReason
}
