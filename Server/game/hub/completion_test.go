package hub_test

import (
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

type goalResult struct {
	Reason   boardmsg.EndReason
	Position board.Position
}

func (r goalResult) EndReason() boardmsg.EndReason { return r.Reason }

type goalRules struct {
	goal board.Position
}

func (goalRules) TurnLimit() uint64 { return 0 }

func (r goalRules) AfterTurn(state board.Snapshot, _ fieldevent.UpdateState) bool {
	return state.Robots["a"].Position() == r.goal
}

func (goalRules) Result(state board.Snapshot, reason boardmsg.EndReason) (boardmsg.CompletionResult, error) {
	return goalResult{Reason: reason, Position: state.Robots["a"].Position()}, nil
}

func TestRunWithCustomCompletion(t *testing.T) {
	t.Parallel()

	goal := board.NewPosition(1, 0)
	want := goalResult{Reason: boardmsg.EndConditionMet, Position: goal}
	config, err := board.NewConfig(
		[][]board.Terrain{{
			board.NewLandTerrain(board.PlainLand{}, false, board.NoTeam),
			board.NewLandTerrain(board.GoalLand{}, false, board.NoTeam),
		}},
		map[protocol.RobotID]board.RobotState{
			"a": board.NewRobotState(board.NewPosition(0, 0), board.East, 0),
		},
		0,
		goalRules{goal: goal},
	)
	if err != nil {
		t.Fatal(err)
	}
	b, err := board.New(config)
	if err != nil {
		t.Fatal(err)
	}
	h := newHub(t, b, hub.Participant{
		ID: "a",
		Robot: robotFunc(func(values.Value) (robotmsg.Step, error) {
			return robotmsg.ActionStep{Action: actions.ForwardAction{}}, nil
		}),
	})

	report, runErr := h.Run(limits())

	if runErr != nil || report.Result() != want {
		t.Errorf("result = %+v, %v, want %+v", report.Result(), runErr, want)
	}
	if report.Resumes() != 1 || report.Outcome("a") != (hub.InterruptedOutcome{}) {
		t.Errorf("unexpected robot outcomes: %+v", report)
	}
}
