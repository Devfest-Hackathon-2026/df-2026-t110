package value

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

func (DistanceTarget) recordValue() {}
