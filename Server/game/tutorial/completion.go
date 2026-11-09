package tutorial

import (
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
)

// TurnLimit is the number of turns allowed for a tutorial attempt.
// It is a normal completion condition, independent of the board's safety limit.
const TurnLimit uint64 = 1_000

type completionRules struct {
	robotID       protocol.RobotID
	condition     string
	checkpoints   []int64
	requiredCount int
}

func newCompletionRules(condition schema.ClearCondition, robotID protocol.RobotID, checkpoints []int64, terrain [][]board.Terrain) (*completionRules, error) {
	rules := &completionRules{robotID: robotID}
	variants := 0
	for _, present := range []bool{condition.ReachGoal != nil, condition.VisitCheckpoints != nil, condition.DeliverLuggage != nil} {
		if present {
			variants++
		}
	}
	if variants != 1 {
		return nil, fmt.Errorf("tutorial clear condition must have exactly one variant")
	}
	switch {
	case condition.ReachGoal != nil:
		rules.condition = schema.ReachGoalType
	case condition.VisitCheckpoints != nil:
		if len(checkpoints) == 0 {
			return nil, fmt.Errorf("visit_checkpoints requires at least one checkpoint")
		}
		rules.condition = schema.VisitCheckpointsType
		rules.checkpoints = checkpoints
	case condition.DeliverLuggage != nil:
		delivery := condition.DeliverLuggage
		rules.condition = schema.DeliverLuggageType
		rules.requiredCount = delivery.RequiredCount
		if rules.requiredCount < 1 {
			return nil, fmt.Errorf("deliver_luggage requires a positive count")
		}
		var destinations, luggage int
		for _, row := range terrain {
			for _, cell := range row {
				if land, ok := cell.(board.LandTerrain); ok {
					if land.TerritoryOwner() != board.NoTeam {
						destinations++
					}
					if land.HasLuggage() {
						luggage++
					}
				}
			}
		}
		if destinations < rules.requiredCount || luggage < rules.requiredCount {
			return nil, fmt.Errorf("deliver_luggage requires %d destination cells and pieces of luggage, got %d and %d", rules.requiredCount, destinations, luggage)
		}
	}

	return rules, nil
}

func (*completionRules) TurnLimit() uint64 { return TurnLimit }

func (r *completionRules) AfterTurn(state board.Snapshot, update fieldevent.UpdateState) bool {
	if r.condition == schema.DeliverLuggageType {
		// Count only luggage placed on territory cells. Carrying
		// luggage over a territory never counts as delivery.
		delivered := 0
		for _, row := range state.Terrain {
			for _, cell := range row {
				if land, ok := cell.(board.LandTerrain); ok && land.TerritoryOwner() != board.NoTeam && land.HasLuggage() {
					delivered++
				}
			}
		}
		return delivered >= r.requiredCount
	}
	for _, action := range update.Actions {
		move, ok := action.(fieldevent.MoveResult)
		if !ok || !move.Success || move.RobotID != r.robotID {
			continue
		}

		robot := state.Robots[r.robotID]
		position := robot.Position()
		// A successful move always ends on land.
		land, ok := state.Terrain[position.Y()][position.X()].(board.LandTerrain)
		if !ok {
			return false
		}
		switch r.condition {
		case schema.ReachGoalType:
			_, goal := land.Kind().(board.GoalLand)
			return goal
		case schema.VisitCheckpointsType:
			checkpoint, ok := land.Kind().(board.CheckpointLand)
			if ok && checkpoint.Order() == r.checkpoints[0] {
				r.checkpoints = r.checkpoints[1:]
				return len(r.checkpoints) == 0
			}
		}
	}
	return false
}

func (*completionRules) Result(state board.Snapshot, reason boardmsg.EndReason) (boardmsg.CompletionResult, error) {
	return boardmsg.NewTutorialResult(state.Tick, reason), nil
}
