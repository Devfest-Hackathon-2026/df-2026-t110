package engine

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func applyDrill(id protocol.RobotID, robot *RobotState, terrain [][]Terrain, target Position, success bool) (values.BoolValue, fieldevent.DrillResult) {
	event := fieldevent.DrillResult{RobotID: id, Success: success}
	if success {
		terrain[target.y][target.x] = NewLandTerrain(PlainLand{}, false, NoTeam)
		robot.drillDurability--
		event.Target = &fieldevent.Position{X: target.x, Y: target.y}
	}
	event.DrillDurability = robot.drillDurability

	return values.BoolValue{Bool: success}, event
}
