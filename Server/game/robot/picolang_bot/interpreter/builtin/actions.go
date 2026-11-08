package builtin

import (
	"errors"
	"fmt"

	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
)

func actionFor(name bot.Name, args []memory.Value) actions.Action {
	switch name {
	case "forward":
		return actions.ForwardAction{}
	case "back":
		return actions.BackAction{}
	case "rotate":
		return actions.RotateAction{Rotation: args[0].(memory.Rotation).Value}
	case "wait":
		return actions.WaitAction{Integer: args[0].(memory.Integer).Value}
	case "pick_up":
		return actions.PickUpAction{}
	case "release":
		return actions.ReleaseAction{}
	case "drill":
		return actions.DrillAction{}
	case "drill_durability":
		return actions.DrillDurabilityAction{}
	case "distance":
		return actions.DistanceAction{Target: args[0].(memory.DistanceTarget).Value}
	case "search_front":
		return actions.SearchFrontAction{}
	case "search_now":
		return actions.SearchNowAction{}
	case "print_integer":
		return actions.PrintIntegerAction{Integer: args[0].(memory.Integer).Value}
	case "print_bool":
		return actions.PrintBoolAction{Bool: args[0].(memory.Bool).Value}
	case "print_unit":
		return actions.PrintUnitAction{}
	case "print_terrain":
		return actions.PrintTerrainAction{Terrain: args[0].(memory.Terrain).Value}
	case "print_rotation":
		return actions.PrintRotationAction{Rotation: args[0].(memory.Rotation).Value}
	case "print_distance_target":
		return actions.PrintDistanceTargetAction{DistanceTarget: args[0].(memory.DistanceTarget).Value}
	default:
		return nil
	}
}

// Complete checks and copies a board result into memory. The returned ID has one
// caller-owned reference. Rejected results do not allocate or consume anything.
// The request must be the Action previously returned by Evaluate.
func (e *Evaluator) Complete(request Action, result values.Value) (memory.ID, error) {
	if e == nil || e.memory == nil || !actions.ValidAction(request.Action) || !values.ValidValue(result) {
		return 0, errors.New("a builtin action and valid board result are required")
	}
	var value memory.Value
	switch v := result.(type) {
	case values.BoolValue:
		if _, ok := request.ResultType.(bot.BoolType); ok {
			value = memory.Bool{Value: v.Bool}
		}
	case values.IntegerValue:
		if _, ok := request.ResultType.(bot.IntegerType); ok {
			value = memory.Integer{Value: v.Integer}
		}
	case values.UnitValue:
		if _, ok := request.ResultType.(bot.UnitType); ok {
			value = memory.Unit{}
		}
	case values.TerrainValue:
		if _, ok := request.ResultType.(bot.TerrainType); ok {
			value = memory.Terrain{Value: v}
		}
	}
	if value == nil {
		return 0, fmt.Errorf("board result %T does not match action result type %s", result, bot.TypeName(request.ResultType))
	}
	return e.memory.Add(value)
}
