package engine

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func (b *Board) distance(id protocol.RobotID, target values.DistanceTarget) int64 {
	robot := b.state.Robots[id]
	delta := directionDeltas[robot.direction]
	switch target {
	case values.DistanceRobot, values.DistanceAlly, values.DistanceEnemy:
		return b.distanceToRobot(robot, delta, target)
	}

	position := robot.position
	for distance := int64(1); ; distance++ {
		position.x += delta.x
		position.y += delta.y
		if !b.contains(position) {
			if target == values.DistanceSea {
				return distance
			}
			return -1
		}
		if matchesTerrainTarget(b.state.Terrain[position.y][position.x], target) {
			return distance
		}
	}
}

func (b *Board) distanceToRobot(robot RobotState, delta Position, target values.DistanceTarget) int64 {
	closest := int64(-1)
	for _, other := range b.state.Robots {
		switch target {
		case values.DistanceAlly:
			if robot.teamID == NoTeam || other.teamID != robot.teamID {
				continue
			}
		case values.DistanceEnemy:
			if robot.teamID == NoTeam || other.teamID == NoTeam || other.teamID == robot.teamID {
				continue
			}
		}

		dx, dy := other.position.x-robot.position.x, other.position.y-robot.position.y
		// A target must lie on the forward ray, excluding the robot's own cell.
		if dx*delta.y != dy*delta.x {
			continue
		}
		distance := int64(dx*delta.x + dy*delta.y)
		if distance > 0 && (closest == -1 || distance < closest) {
			closest = distance
		}
	}
	return closest
}

func matchesTerrainTarget(terrain Terrain, target values.DistanceTarget) bool {
	switch cell := terrain.(type) {
	case RockTerrain:
		return target == values.DistanceRock
	case SeaTerrain:
		return target == values.DistanceSea
	case LandTerrain:
		switch target {
		case values.DistanceLuggage:
			return cell.hasLuggage
		case values.DistanceGoal:
			_, goal := cell.kind.(GoalLand)
			return goal
		case values.DistanceTerritory:
			return cell.territoryOwner != NoTeam
		}
	}
	return false
}
