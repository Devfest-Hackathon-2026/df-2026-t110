package field_event

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

type PrintUnitResult struct {
	RobotID protocol.RobotID
	ActionResult
}

type PrintTerrainResult struct {
	RobotID protocol.RobotID
	Terrain values.TerrainValue
	ActionResult
}

type PrintRotationResult struct {
	RobotID  protocol.RobotID
	Rotation values.Rotation
	ActionResult
}

type PrintDistanceTargetResult struct {
	RobotID        protocol.RobotID
	DistanceTarget values.DistanceTarget
	ActionResult
}
