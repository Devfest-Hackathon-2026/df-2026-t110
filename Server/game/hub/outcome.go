package hub

type Outcome interface{ outcome() }

type UnstartedOutcome struct{}
type StoppedOutcome struct{}
type CompletedOutcome struct{}
type FailedOutcome struct{ Err error }

type outcomeMarker = Outcome

// InterruptedOutcome identifies a robot whose program was still active when the game ended.
type InterruptedOutcome struct{ outcomeMarker }

func (UnstartedOutcome) outcome() {}
func (StoppedOutcome) outcome()   {}
func (CompletedOutcome) outcome() {}
func (FailedOutcome) outcome()    {}
