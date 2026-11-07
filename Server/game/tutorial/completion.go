package tutorial

import (
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
)

type completionRules struct {
	robotID     protocol.RobotID
	condition   string
	checkpoints []int64
}

func newCompletionRules(condition schema.ClearCondition, robotID protocol.RobotID, checkpoints []int64) (*completionRules, error) {
	rules := &completionRules{robotID: robotID}
	switch {
	case condition.ReachGoal != nil && condition.VisitCheckpoints == nil:
		rules.condition = schema.ReachGoalType
	case condition.VisitCheckpoints != nil && condition.ReachGoal == nil:
		if len(checkpoints) == 0 {
			return nil, fmt.Errorf("visit_checkpoints requires at least one checkpoint")
		}
		rules.condition = schema.VisitCheckpointsType
		rules.checkpoints = checkpoints
	default:
		return nil, fmt.Errorf("tutorial clear condition must have exactly one variant")
	}

	return rules, nil
}

func (*completionRules) TurnLimit() uint64 { return 0 }

func (r *completionRules) AfterTurn(state board.Snapshot, update fieldevent.UpdateState) bool {
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
