package engine

import (
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
)

// CompletionRules configure one board run. The board always ends when all robots stop.
type CompletionRules interface {
	// TurnLimit sets a normal end at this tick. Zero disables the limit.
	TurnLimit() uint64
	// AfterTurn can request completion using the committed board and update, before
	// any robot resumes. Idle ticks may be batched. Treat the update as read-only.
	AfterTurn(Snapshot, fieldevent.UpdateState) bool
	// Result builds the mode-specific result from the final snapshot before cleanup.
	Result(Snapshot, boardmsg.EndReason) (boardmsg.CompletionResult, error)
}

// AllStoppedRules ends the run when every robot has stopped.
type AllStoppedRules struct{}

func (AllStoppedRules) TurnLimit() uint64 { return 0 }

func (AllStoppedRules) AfterTurn(Snapshot, fieldevent.UpdateState) bool { return false }

func (AllStoppedRules) Result(Snapshot, boardmsg.EndReason) (boardmsg.CompletionResult, error) {
	return nil, nil
}

func (b *Board) finish(output boardmsg.BoardOutput, reason boardmsg.EndReason) (boardmsg.BoardOutput, error) {
	result, err := b.completion.Result(b.Snapshot(), reason)
	if err != nil {
		b.phase = finishedBoard{err: err}
		return output, err
	}
	decision := boardmsg.NewDoneDecision(result, reason != boardmsg.EndAllStopped)
	if reason == boardmsg.EndAllStopped {
		var update fieldevent.UpdateState
		for _, id := range b.order {
			if _, present := b.state.Robots[id]; present {
				update.RemovedRobotIDs = append(update.RemovedRobotIDs, id)
			}
		}
		clear(b.state.Robots)
		output.Events = append(output.Events, fieldevent.FieldEvent{Tick: b.state.Tick, State: update})
	}
	output.Decision = decision
	b.phase = finishedBoard{decision: decision}
	return output, nil
}
