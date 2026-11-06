package game

// Action covers the action builtins in Docs/picolang.md. Each concrete type
// carries only its builtin's arguments. Tick costs and execution belong to the
// board. The unexported method restricts implementations to this package.
type Action interface {
	action()
}

type ForwardAction struct{}
type BackAction struct{}
type RotateAction struct{ Rotation Rotation }
type WaitAction struct{ Integer int64 }
type PickUpAction struct{}
type ReleaseAction struct{}
type DrillAction struct{}
type DrillDurabilityAction struct{}
type DistanceAction struct{}
type SearchFrontAction struct{}
type SearchNowAction struct{}
type PrintIntegerAction struct{ Integer int64 }
type PrintBoolAction struct{ Bool bool }

func (ForwardAction) action()         {}
func (BackAction) action()            {}
func (RotateAction) action()          {}
func (WaitAction) action()            {}
func (PickUpAction) action()          {}
func (ReleaseAction) action()         {}
func (DrillAction) action()           {}
func (DrillDurabilityAction) action() {}
func (DistanceAction) action()        {}
func (SearchFrontAction) action()     {}
func (SearchNowAction) action()       {}
func (PrintIntegerAction) action()    {}
func (PrintBoolAction) action()       {}

// ValidAction rejects nil and pointer variants at protocol boundaries.
func ValidAction(action Action) bool {
	switch action.(type) {
	case ForwardAction, BackAction, RotateAction, WaitAction, PickUpAction,
		ReleaseAction, DrillAction, DrillDurabilityAction, DistanceAction,
		SearchFrontAction, SearchNowAction, PrintIntegerAction, PrintBoolAction:
		return true
	default:
		return false
	}
}
