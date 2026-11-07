package board

// TutorialResult describes the outcome of a tutorial board run.
type TutorialResult struct {
	tick   uint64
	reason EndReason
}

func NewTutorialResult(tick uint64, reason EndReason) TutorialResult {
	return TutorialResult{tick: tick, reason: reason}
}

func (r TutorialResult) Tick() uint64         { return r.tick }
func (r TutorialResult) EndReason() EndReason { return r.reason }
func (r TutorialResult) Cleared() bool        { return r.reason == EndConditionMet }
