package action

import (
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

// Action covers the action builtins in Docs/picolang.md. Each concrete type
// carries only its builtin's arguments. Tick costs and execution belong to the
// board. The unexported method restricts implementations to this package.
type Action interface {
	action()
}

type ForwardAction struct{}
type BackAction struct{}
type RotateAction struct{ Rotation values.Rotation }
type WaitAction struct{ Integer int64 }
type PickUpAction struct{}
type ReleaseAction struct{}
type DrillAction struct{}
type DrillDurabilityAction struct{}
type DistanceAction struct{ Target DistanceTarget }
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

type DistanceTarget string

const (
	DistanceRobot    DistanceTarget = "robot"
	DistanceAlly     DistanceTarget = "ally"
	DistanceEnemy    DistanceTarget = "enemy"
	DistanceRock     DistanceTarget = "rock"
	DistanceLuggage  DistanceTarget = "luggage"
	DistanceSea      DistanceTarget = "sea"
	DistanceGoal     DistanceTarget = "goal"
	DistanceGoalArea DistanceTarget = "goal_area"
)

func (target DistanceTarget) Valid() bool {
	switch target {
	case DistanceRobot, DistanceAlly, DistanceEnemy, DistanceRock, DistanceLuggage,
		DistanceSea, DistanceGoal, DistanceGoalArea:
		return true
	default:
		return false
	}
}

// ValidAction rejects nil, pointer variants, and invalid distance targets.
func ValidAction(action Action) bool {
	switch action := action.(type) {
	case ForwardAction, BackAction, RotateAction, WaitAction, PickUpAction,
		ReleaseAction, DrillAction, DrillDurabilityAction,
		SearchFrontAction, SearchNowAction, PrintIntegerAction, PrintBoolAction:
		return true
	case DistanceAction:
		return action.Target.Valid()
	default:
		return false
	}
}
