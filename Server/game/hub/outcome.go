package hub

type Outcome interface{ outcome() }

type UnstartedOutcome struct{}
type StoppedOutcome struct{}
type CompletedOutcome struct{}
type FailedOutcome struct{ Err error }

func (UnstartedOutcome) outcome() {}
func (StoppedOutcome) outcome()   {}
func (CompletedOutcome) outcome() {}
func (FailedOutcome) outcome()    {}
