package tutorialrun

import (
	"fmt"
	"math"

	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	wire "github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/history"
)

// Union slots not supported by the generator remain interfaces, but each event
// and action is constructed with a generated struct rather than a string map.
func historyEvent(event gameevent.GameEvent) (any, error) {
	switch event := event.(type) {
	case gameevent.FieldEvent:
		if event.Event.Tick > uint64(math.MaxInt) {
			return nil, fmt.Errorf("history tick exceeds integer range")
		}
		state, err := fieldState(event.Event.State)
		return wire.FieldEvent{Type: "field", Tick: int(event.Event.Tick), State: state}, err
	case gameevent.RobotEvent:
		debug, err := debugEvent(event.Event)
		return wire.PicolangEvent{Type: "picolang", InterpreterId: string(event.InterpreterID), Event: debug}, err
	default:
		return nil, fmt.Errorf("unsupported history event %T", event)
	}
}

func ptr[T any](value T) *T { return &value }
func position(p *fieldevent.Position) *wire.Position {
	if p == nil {
		return nil
	}
	return &wire.Position{X: p.X, Y: p.Y}
}
func fieldState(state fieldevent.FieldState) (wire.FieldEventState, error) {
	switch state := state.(type) {
	case fieldevent.InitialState:
		terrain := make([][]wire.Cell, len(state.Terrain))
		for y, row := range state.Terrain {
			terrain[y] = make([]wire.Cell, len(row))
			for x, cell := range row {
				out := wire.Cell{Type: wire.Terrain(cell.Type)}
				if cell.Kind != nil {
					kind := &wire.LandKind{Type: wire.LandKindType(cell.Kind.Type)}
					if cell.Kind.Order != nil {
						kind.Order = ptr(int(*cell.Kind.Order))
					}
					out.Kind = kind
				}
				if cell.HasLuggage {
					out.HasLuggage = ptr(true)
				}
				if cell.TerritoryOwner != 0 {
					out.TerritoryOwner = ptr(cell.TerritoryOwner)
				}
				terrain[y][x] = out
			}
		}
		robots := make([]wire.RobotState, 0, len(state.Robots))
		for _, robot := range state.Robots {
			out := wire.RobotState{Id: wire.RobotId(robot.ID), Position: *position(&robot.Position), Direction: wire.Direction(robot.Direction), DrillDurability: int(robot.DrillDurability)}
			if robot.TeamID != 0 {
				out.TeamId = ptr(wire.RobotStateTeamId(robot.TeamID))
			}
			if robot.CarryingLuggage {
				out.CarryingLuggage = ptr(true)
			}
			robots = append(robots, out)
		}
		return wire.FieldEventState{Initial: &wire.InitialState{Kind: "initial", Terrain: terrain, Robots: robots}}, nil
	case fieldevent.UpdateState:
		out := wire.UpdateState{Kind: "update"}
		for _, action := range state.Actions {
			value, err := actionResult(action)
			if err != nil {
				return wire.FieldEventState{}, err
			}
			out.Actions = append(out.Actions, value)
		}
		for _, id := range state.RemovedRobotIDs {
			out.RemovedRobotIds = append(out.RemovedRobotIds, wire.RobotId(id))
		}
		return wire.FieldEventState{Update: &out}, nil
	default:
		return wire.FieldEventState{}, fmt.Errorf("unsupported field state %T", state)
	}
}
func actionResult(action fieldevent.ActionResult) (any, error) {
	switch a := action.(type) {
	case fieldevent.MoveResult:
		return wire.MoveResult{Kind: wire.MoveResultKind(a.Kind), RobotId: wire.RobotId(a.RobotID), Success: a.Success, To: position(a.To)}, nil
	case fieldevent.RotateResult:
		return wire.RotateResult{Kind: "rotate", RobotId: wire.RobotId(a.RobotID), Rotation: wire.RotateResultRotation(a.Rotation), Direction: wire.Direction(a.Direction)}, nil
	case fieldevent.DrillResult:
		return wire.DrillResult{Kind: "drill", RobotId: wire.RobotId(a.RobotID), Success: a.Success, Target: position(a.Target), DrillDurability: int(a.DrillDurability)}, nil
	case fieldevent.PickUpResult:
		return wire.PickUpResult{Kind: "pick_up", RobotId: wire.RobotId(a.RobotID), Success: a.Success, Target: position(a.Target)}, nil
	case fieldevent.ReleaseResult:
		out := wire.ReleaseResult{Kind: "release", RobotId: wire.RobotId(a.RobotID), Success: a.Success, Target: position(a.Target)}
		if a.Discarded {
			out.Discarded = ptr(true)
		}
		return out, nil
	case fieldevent.WaitResult:
		return wire.WaitResult{Kind: "wait", RobotId: wire.RobotId(a.RobotID), Ticks: int(a.Ticks)}, nil
	case fieldevent.PrintIntegerResult:
		return wire.PrintIntegerResult{Kind: "print_integer", RobotId: wire.RobotId(a.RobotID), Integer: int(a.Integer)}, nil
	case fieldevent.PrintBoolResult:
		return wire.PrintBoolResult{Kind: "print_bool", RobotId: wire.RobotId(a.RobotID), Bool: a.Bool}, nil
	case fieldevent.PrintUnitResult:
		return wire.PrintUnitResult{Kind: "print_unit", RobotId: wire.RobotId(a.RobotID)}, nil
	case fieldevent.PrintTerrainResult:
		terrain, err := terrainValue(a.Terrain)
		return wire.PrintTerrainResult{Kind: "print_terrain", RobotId: wire.RobotId(a.RobotID), Terrain: terrain}, err
	case fieldevent.PrintRotationResult:
		return wire.PrintRotationResult{Kind: "print_rotation", RobotId: wire.RobotId(a.RobotID), Rotation: wire.PrintRotationResultRotation(a.Rotation)}, nil
	case fieldevent.PrintDistanceTargetResult:
		return wire.PrintDistanceTargetResult{Kind: "print_distance_target", RobotId: wire.RobotId(a.RobotID), DistanceTarget: wire.DistanceTarget(a.DistanceTarget)}, nil
	default:
		return nil, fmt.Errorf("unsupported action result %T", action)
	}
}
func debugEvent(event robotevent.PicolangEvent) (wire.PicolangDebugEvent, error) {
	switch e := event.(type) {
	case robotevent.CurrentNodeEvent:
		return wire.PicolangDebugEvent{CurrentNode: &wire.PicolangCurrentNodeEvent{Kind: "current_node", HintId: wire.HintId(e.HintID)}}, nil
	case robotevent.VariableInitEvent:
		value, err := recordValue(e.Value)
		return wire.PicolangDebugEvent{VariableInit: &wire.PicolangVariableInitEvent{Kind: "variable_init", HintId: wire.HintId(e.HintID), Value: value}}, err
	case robotevent.VariableDropEvent:
		return wire.PicolangDebugEvent{VariableDrop: &wire.PicolangVariableDropEvent{Kind: "variable_drop", HintId: wire.HintId(e.HintID)}}, nil
	case robotevent.ExecutionFinishedEvent:
		return wire.PicolangDebugEvent{ExecutionFinished: &wire.PicolangExecutionFinishedEvent{Kind: "execution_finished", HintId: wire.HintId(e.HintID), Reason: wire.PicolangExecutionFinishedEventReason(e.Reason), Message: e.Message}}, nil
	case robotevent.VariableUpdateEvent:
		diff := make([]wire.Update, 0, len(e.Diff))
		for _, update := range e.Diff {
			value, err := valueUpdate(update)
			if err != nil {
				return wire.PicolangDebugEvent{}, err
			}
			diff = append(diff, value)
		}
		return wire.PicolangDebugEvent{VariableUpdate: &wire.PicolangVariableUpdateEvent{Kind: "variable_update", HintId: wire.HintId(e.HintID), Diff: diff}}, nil
	default:
		return wire.PicolangDebugEvent{}, fmt.Errorf("unsupported debug event %T", event)
	}
}
func terrainValue(value values.TerrainValue) (wire.Terrain_1, error) {
	switch value := value.(type) {
	case values.RockTerrain:
		return wire.Terrain_1{Rock: &wire.Rock{Type: "rock"}}, nil
	case values.SeaTerrain:
		return wire.Terrain_1{Sea: &wire.Sea{Type: "sea"}}, nil
	case values.LandTerrain:
		out := wire.Land{Type: "land"}
		switch land := value.Kind.(type) {
		case values.PlainLand:
			out.Kind.Plain = &wire.PlainLand{Type: "plain"}
		case values.GoalLand:
			out.Kind.Goal = &wire.GoalLand{Type: "goal"}
		case values.CheckpointLand:
			out.Kind.Checkpoint = &wire.CheckpointLand{Type: "checkpoint", Order: int(land.Order)}
		default:
			return wire.Terrain_1{}, fmt.Errorf("unsupported land kind %T", value.Kind)
		}
		if value.HasLuggage {
			out.HasLuggage = ptr(true)
		}
		if value.Territory != values.NoTerritory {
			out.Territory = ptr(wire.LandTerritory(value.Territory))
		}
		return wire.Terrain_1{Land: &out}, nil
	default:
		return wire.Terrain_1{}, fmt.Errorf("unsupported terrain %T", value)
	}
}
func recordValue(value values.RecordValue) (wire.Value, error) {
	switch value := value.(type) {
	case values.BoolValue:
		return value.Bool, nil
	case values.IntegerValue:
		return value.Integer, nil
	case values.UnitValue:
		return nil, nil
	case values.RotationRecordValue:
		return string(value.Rotation), nil
	case values.DistanceTarget:
		return string(value), nil
	case values.TerrainValue:
		return terrainValue(value)
	case values.StackRecordValue:
		out := make([]wire.Value, 0, len(value.Elements))
		for _, element := range value.Elements {
			entry, err := recordValue(element)
			if err != nil {
				return nil, err
			}
			out = append(out, entry)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported recorded value %T", value)
	}
}
func valueUpdate(update robotevent.ValueUpdate) (wire.Update, error) {
	switch u := update.(type) {
	case robotevent.BoolUpdate:
		return wire.BoolUpdate{Type: "bool", Value: u.Value}, nil
	case robotevent.IntegerUpdate:
		return wire.IntegerUpdate{Type: "integer", Value: int(u.Value)}, nil
	case robotevent.UnitUpdate:
		return wire.UnitUpdate{Type: "unit", Value: nil}, nil
	case robotevent.RotationUpdate:
		return wire.RotationUpdate{Type: "rotation", Value: wire.RotationUpdateValue(u.Value)}, nil
	case robotevent.DistanceTargetUpdate:
		return wire.DistanceTargetUpdate{Type: "distance_target", Value: wire.DistanceTarget_1(u.Value)}, nil
	case robotevent.TerrainUpdate:
		terrain, err := terrainValue(u.Value)
		return wire.TerrainUpdate{Type: "terrain", Value: terrain}, err
	case robotevent.StackUpdate:
		var operation wire.StackUpdateUpdate
		switch op := u.Update.(type) {
		case robotevent.StackReplace:
			values := make([]wire.Value, 0, len(op.Value))
			for _, value := range op.Value {
				entry, err := recordValue(value)
				if err != nil {
					return nil, err
				}
				values = append(values, entry)
			}
			operation.Replace = &wire.StackReplace{Op: "replace", Value: values}
		case robotevent.PushBack:
			value, err := recordValue(op.Value)
			if err != nil {
				return nil, err
			}
			operation.PushBack = &wire.PushBack{Op: "push_back", Value: value}
		case robotevent.PopBack:
			operation.PopBack = &wire.PopBack{Op: "pop_back"}
		default:
			return nil, fmt.Errorf("unsupported stack operation %T", op)
		}
		return wire.StackUpdate{Type: "stack", Update: operation}, nil
	default:
		return nil, fmt.Errorf("unsupported value update %T", update)
	}
}
