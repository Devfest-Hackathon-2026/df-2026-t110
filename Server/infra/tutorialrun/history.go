package tutorialrun

import (
	"fmt"

	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

// The engine uses interface unions without JSON tags. Translate them at the
// infrastructure boundary to the shared game_event schema, preserving i64s.
type object = map[string]any

func historyEvent(event gameevent.GameEvent) (object, error) {
	switch event := event.(type) {
	case gameevent.FieldEvent:
		state, err := fieldState(event.Event.State)
		return object{"type": "field", "tick": event.Event.Tick, "state": state}, err
	case gameevent.RobotEvent:
		debug, err := debugEvent(event.Event)
		return object{"type": "picolang", "interpreterId": event.InterpreterID, "event": debug}, err
	default:
		return nil, fmt.Errorf("unsupported history event %T", event)
	}
}

func position(p *fieldevent.Position) any {
	if p == nil {
		return nil
	}
	return object{"x": p.X, "y": p.Y}
}

func fieldState(state fieldevent.FieldState) (object, error) {
	switch state := state.(type) {
	case fieldevent.InitialState:
		terrain := make([][]object, len(state.Terrain))
		for y, row := range state.Terrain {
			terrain[y] = make([]object, len(row))
			for x, cell := range row {
				out := object{"type": cell.Type}
				if cell.Kind != nil {
					kind := object{"type": cell.Kind.Type}
					if cell.Kind.Order != nil {
						kind["order"] = *cell.Kind.Order
					}
					out["kind"] = kind
				}
				if cell.HasLuggage {
					out["hasLuggage"] = true
				}
				if cell.TerritoryOwner != 0 {
					out["territoryOwner"] = cell.TerritoryOwner
				}
				terrain[y][x] = out
			}
		}
		robots := make([]object, 0, len(state.Robots))
		for _, robot := range state.Robots {
			out := object{"id": robot.ID, "position": position(&robot.Position), "direction": robot.Direction, "drillDurability": robot.DrillDurability}
			if robot.TeamID != 0 {
				out["teamId"] = robot.TeamID
			}
			if robot.CarryingLuggage {
				out["carryingLuggage"] = true
			}
			robots = append(robots, out)
		}
		return object{"kind": "initial", "terrain": terrain, "robots": robots}, nil
	case fieldevent.UpdateState:
		out := object{"kind": "update"}
		if len(state.Actions) != 0 {
			actions := make([]object, 0, len(state.Actions))
			for _, action := range state.Actions {
				value, err := actionResult(action)
				if err != nil {
					return nil, err
				}
				actions = append(actions, value)
			}
			out["actions"] = actions
		}
		if len(state.RemovedRobotIDs) != 0 {
			out["removedRobotIds"] = state.RemovedRobotIDs
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported field state %T", state)
	}
}

func actionResult(action fieldevent.ActionResult) (object, error) {
	var out object
	switch action := action.(type) {
	case fieldevent.MoveResult:
		out = object{"kind": action.Kind, "robotId": action.RobotID, "success": action.Success}
		if action.To != nil {
			out["to"] = position(action.To)
		}
	case fieldevent.RotateResult:
		out = object{"kind": "rotate", "robotId": action.RobotID, "rotation": action.Rotation, "direction": action.Direction}
	case fieldevent.DrillResult:
		out = object{"kind": "drill", "robotId": action.RobotID, "success": action.Success, "drillDurability": action.DrillDurability}
		if action.Target != nil {
			out["target"] = position(action.Target)
		}
	case fieldevent.PickUpResult:
		out = object{"kind": "pick_up", "robotId": action.RobotID, "success": action.Success}
		if action.Target != nil {
			out["target"] = position(action.Target)
		}
	case fieldevent.ReleaseResult:
		out = object{"kind": "release", "robotId": action.RobotID, "success": action.Success}
		if action.Target != nil {
			out["target"] = position(action.Target)
		}
		if action.Discarded {
			out["discarded"] = true
		}
	case fieldevent.WaitResult:
		out = object{"kind": "wait", "robotId": action.RobotID, "ticks": action.Ticks}
	case fieldevent.PrintIntegerResult:
		out = object{"kind": "print_integer", "robotId": action.RobotID, "integer": action.Integer}
	case fieldevent.PrintBoolResult:
		out = object{"kind": "print_bool", "robotId": action.RobotID, "bool": action.Bool}
	case fieldevent.PrintUnitResult:
		out = object{"kind": "print_unit", "robotId": action.RobotID}
	case fieldevent.PrintTerrainResult:
		terrain, err := recordValue(action.Terrain)
		if err != nil {
			return nil, err
		}
		out = object{"kind": "print_terrain", "robotId": action.RobotID, "terrain": terrain}
	case fieldevent.PrintRotationResult:
		out = object{"kind": "print_rotation", "robotId": action.RobotID, "rotation": action.Rotation}
	case fieldevent.PrintDistanceTargetResult:
		out = object{"kind": "print_distance_target", "robotId": action.RobotID, "distanceTarget": action.DistanceTarget}
	default:
		return nil, fmt.Errorf("unsupported action result %T", action)
	}
	return out, nil
}

func debugEvent(event robotevent.PicolangEvent) (object, error) {
	switch event := event.(type) {
	case robotevent.CurrentNodeEvent:
		return object{"kind": "current_node", "hintId": event.HintID}, nil
	case robotevent.VariableInitEvent:
		value, err := recordValue(event.Value)
		return object{"kind": "variable_init", "hintId": event.HintID, "value": value}, err
	case robotevent.VariableDropEvent:
		return object{"kind": "variable_drop", "hintId": event.HintID}, nil
	case robotevent.VariableUpdateEvent:
		diff := make([]object, 0, len(event.Diff))
		for _, update := range event.Diff {
			value, err := valueUpdate(update)
			if err != nil {
				return nil, err
			}
			diff = append(diff, value)
		}
		return object{"kind": "variable_update", "hintId": event.HintID, "diff": diff}, nil
	case robotevent.ExecutionFinishedEvent:
		return object{"kind": "execution_finished", "hintId": event.HintID, "reason": event.Reason, "message": event.Message}, nil
	default:
		return nil, fmt.Errorf("unsupported debug event %T", event)
	}
}

func recordValue(value values.RecordValue) (any, error) {
	switch value := value.(type) {
	case values.BoolValue:
		return value.Bool, nil
	case values.IntegerValue:
		return value.Integer, nil
	case values.UnitValue:
		return nil, nil
	case values.RotationRecordValue:
		return value.Rotation, nil
	case values.DistanceTarget:
		return value, nil
	case values.StackRecordValue:
		out := make([]any, 0, len(value.Elements))
		for _, element := range value.Elements {
			entry, err := recordValue(element)
			if err != nil {
				return nil, err
			}
			out = append(out, entry)
		}
		return out, nil
	case values.RockTerrain:
		return object{"type": "rock"}, nil
	case values.SeaTerrain:
		return object{"type": "sea"}, nil
	case values.LandTerrain:
		var kind object
		switch land := value.Kind.(type) {
		case values.PlainLand:
			kind = object{"type": "plain"}
		case values.GoalLand:
			kind = object{"type": "goal"}
		case values.CheckpointLand:
			kind = object{"type": "checkpoint", "order": land.Order}
		default:
			return nil, fmt.Errorf("unsupported land kind %T", value.Kind)
		}
		out := object{"type": "land", "kind": kind}
		if value.HasLuggage {
			out["hasLuggage"] = true
		}
		if value.Territory != values.NoTerritory {
			out["territory"] = value.Territory
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported recorded value %T", value)
	}
}

func valueUpdate(update robotevent.ValueUpdate) (object, error) {
	var kind string
	var value values.RecordValue
	switch update := update.(type) {
	case robotevent.BoolUpdate:
		kind, value = "bool", values.BoolValue{Bool: update.Value}
	case robotevent.IntegerUpdate:
		kind, value = "integer", values.IntegerValue{Integer: update.Value}
	case robotevent.UnitUpdate:
		kind, value = "unit", values.UnitValue{}
	case robotevent.TerrainUpdate:
		kind, value = "terrain", update.Value
	case robotevent.RotationUpdate:
		kind, value = "rotation", values.RotationRecordValue{Rotation: update.Value}
	case robotevent.DistanceTargetUpdate:
		kind, value = "distance_target", update.Value
	case robotevent.StackUpdate:
		var operation object
		switch update := update.Update.(type) {
		case robotevent.StackReplace:
			value, err := recordValue(values.StackRecordValue{Elements: update.Value})
			if err != nil {
				return nil, err
			}
			operation = object{"op": "replace", "value": value}
		case robotevent.PushBack:
			value, err := recordValue(update.Value)
			if err != nil {
				return nil, err
			}
			operation = object{"op": "push_back", "value": value}
		case robotevent.PopBack:
			operation = object{"op": "pop_back"}
		default:
			return nil, fmt.Errorf("unsupported stack operation %T", update)
		}
		return object{"type": "stack", "update": operation}, nil
	default:
		return nil, fmt.Errorf("unsupported value update %T", update)
	}
	encoded, err := recordValue(value)
	return object{"type": kind, "value": encoded}, err
}
