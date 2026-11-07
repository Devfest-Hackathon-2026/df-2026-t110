package engine

import values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"

func drillDurability(robot RobotState) values.IntegerValue {
	return values.IntegerValue{Integer: robot.drillDurability}
}
