package value

type DistanceTarget string

const (
	DistanceRobot     DistanceTarget = "robot"
	DistanceAlly      DistanceTarget = "ally"
	DistanceEnemy     DistanceTarget = "enemy"
	DistanceRock      DistanceTarget = "rock"
	DistanceLuggage   DistanceTarget = "luggage"
	DistanceSea       DistanceTarget = "sea"
	DistanceGoal      DistanceTarget = "goal"
	DistanceTerritory DistanceTarget = "territory"
)

func (target DistanceTarget) Valid() bool {
	switch target {
	case DistanceRobot, DistanceAlly, DistanceEnemy, DistanceRock, DistanceLuggage,
		DistanceSea, DistanceGoal, DistanceTerritory:
		return true
	default:
		return false
	}
}
