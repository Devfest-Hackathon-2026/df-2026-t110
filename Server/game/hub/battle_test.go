package hub_test

import (
	"errors"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func TestBattleTurnLimitScoresTheLastTurn(t *testing.T) {
	t.Parallel()

	plain := board.NewLandTerrain(board.PlainLand{}, false, board.NoTeam)
	luggage := board.NewLandTerrain(board.PlainLand{}, true, board.NoTeam)
	territory := board.NewLandTerrain(board.PlainLand{}, false, board.TeamID2)
	want := boardmsg.NewBattleResult(
		3, boardmsg.EndTurnLimit,
		boardmsg.NewTeamResult(0, 1, -2),
		boardmsg.NewTeamResult(1, 0, 5),
		int(board.TeamID2),
	)
	rules, err := board.NewBattleConfig(3, 5, 2).CompletionRules()
	if err != nil {
		t.Fatal(err)
	}
	config, err := board.NewConfig(
		[][]board.Terrain{{plain, luggage, plain, plain, luggage, territory}},
		map[protocol.RobotID]board.RobotState{
			"a": board.NewRobotState(board.NewPosition(0, 0), board.East, 0).WithTeam(board.TeamID1),
			"b": board.NewRobotState(board.NewPosition(3, 0), board.East, 0).WithTeam(board.TeamID1),
			"c": board.NewRobotState(board.NewPosition(5, 0), board.West, 0).WithTeam(board.TeamID2),
		},
		3,
		rules,
	)
	if err != nil {
		t.Fatal(err)
	}
	b, err := board.New(config)
	if err != nil {
		t.Fatal(err)
	}
	script := func(commands ...actions.Action) robotFunc {
		return func(values.Value) (robotmsg.Step, error) {
			if len(commands) == 0 {
				t.Error("robot resumed after the last turn")
				return robotmsg.DoneStep{}, nil
			}
			action := commands[0]
			commands = commands[1:]
			return robotmsg.ActionStep{Action: action}, nil
		}
	}
	h := newHub(t, b,
		hub.Participant{ID: "a", Robot: script(actions.PickUpAction{}, actions.RotateAction{Rotation: values.Left}, actions.ReleaseAction{})},
		hub.Participant{ID: "b", Robot: script(actions.PickUpAction{}, actions.ForwardAction{}, actions.ReleaseAction{})},
		hub.Participant{ID: "c", Robot: doneRobot()},
	)

	report, runErr := h.Run(limits())
	finished := b.Snapshot()
	repeated, repeatErr := b.Next(nil)

	if runErr != nil || report.Result() != want {
		t.Errorf("battle = %+v, %v, want %+v", report.Result(), runErr, want)
	}
	if report.Resumes() != 7 || report.Outcome("a") != (hub.InterruptedOutcome{}) || report.Outcome("b") != (hub.InterruptedOutcome{}) || report.Outcome("c") != (hub.CompletedOutcome{}) {
		t.Errorf("unexpected robot outcomes: %+v", report)
	}
	if finished.Tick != 3 || finished.Terrain[0][5] != board.NewLandTerrain(board.PlainLand{}, true, board.TeamID2) {
		t.Errorf("final placement was not committed: %+v", finished)
	}
	if repeatErr != nil || repeated.Decision != boardmsg.NewDoneDecision(want, true) || len(repeated.Events) != 0 {
		t.Errorf("repeated completion = %+v, %v", repeated, repeatErr)
	}
}

func TestBattleEndsWhenAllRobotsStop(t *testing.T) {
	t.Parallel()

	plain := board.NewLandTerrain(board.PlainLand{}, false, board.NoTeam)
	failure := errors.New("execution failed")
	want := boardmsg.NewBattleResult(0, boardmsg.EndAllStopped, boardmsg.TeamResult{}, boardmsg.TeamResult{}, 0)
	rules, err := board.NewBattleConfig(10, 1, 1).CompletionRules()
	if err != nil {
		t.Fatal(err)
	}
	config, err := board.NewConfig(
		[][]board.Terrain{{plain, plain}},
		map[protocol.RobotID]board.RobotState{
			"a": board.NewRobotState(board.NewPosition(0, 0), board.East, 0).WithTeam(board.TeamID1),
			"b": board.NewRobotState(board.NewPosition(1, 0), board.West, 0).WithTeam(board.TeamID2),
		},
		0,
		rules,
	)
	if err != nil {
		t.Fatal(err)
	}
	b, err := board.New(config)
	if err != nil {
		t.Fatal(err)
	}
	h := newHub(t, b,
		hub.Participant{ID: "a", Robot: doneRobot()},
		hub.Participant{ID: "b", Robot: robotFunc(func(values.Value) (robotmsg.Step, error) { return nil, failure })},
	)

	report, runErr := h.Run(limits())
	finished := b.Snapshot()

	if runErr != nil || report.Result() != want || len(finished.Robots) != 0 {
		t.Errorf("battle = %+v, %v; board = %+v", report.Result(), runErr, finished)
	}
	if report.Outcome("a") != (hub.CompletedOutcome{}) || report.Outcome("b") != (hub.FailedOutcome{Err: failure}) {
		t.Errorf("unexpected robot outcomes: %+v", report)
	}
}

func TestBattleEndsWhenAllLuggageIsDiscarded(t *testing.T) {
	t.Parallel()

	plain := board.NewLandTerrain(board.PlainLand{}, false, board.NoTeam)
	luggage := board.NewLandTerrain(board.PlainLand{}, true, board.NoTeam)
	want := boardmsg.NewBattleResult(
		4, boardmsg.EndAllLuggageDiscarded,
		boardmsg.NewTeamResult(0, 1, -2),
		boardmsg.NewTeamResult(0, 1, -2),
		0,
	)
	rules, err := board.NewBattleConfig(10, 5, 2).CompletionRules()
	if err != nil {
		t.Fatal(err)
	}
	config, err := board.NewConfig(
		[][]board.Terrain{{plain, luggage, luggage, plain}},
		map[protocol.RobotID]board.RobotState{
			"a": board.NewRobotState(board.NewPosition(0, 0), board.East, 0).WithTeam(board.TeamID1),
			"b": board.NewRobotState(board.NewPosition(3, 0), board.West, 0).WithTeam(board.TeamID2),
		},
		0,
		rules,
	)
	if err != nil {
		t.Fatal(err)
	}
	b, err := board.New(config)
	if err != nil {
		t.Fatal(err)
	}
	script := func(commands ...actions.Action) robotFunc {
		return func(values.Value) (robotmsg.Step, error) {
			if len(commands) == 0 {
				t.Error("robot resumed after the final discard")
				return robotmsg.DoneStep{}, nil
			}
			action := commands[0]
			commands = commands[1:]
			return robotmsg.ActionStep{Action: action}, nil
		}
	}
	h := newHub(t, b,
		hub.Participant{ID: "a", Robot: script(
			actions.PickUpAction{}, actions.RotateAction{Rotation: values.Left},
			actions.ReleaseAction{}, actions.WaitAction{Integer: 1},
		)},
		hub.Participant{ID: "b", Robot: script(
			actions.PickUpAction{}, actions.RotateAction{Rotation: values.Right},
			actions.WaitAction{Integer: 1}, actions.ReleaseAction{},
		)},
	)

	report, runErr := h.Run(limits())

	if runErr != nil || report.Result() != want {
		t.Errorf("battle = %+v, %v, want %+v", report.Result(), runErr, want)
	}
	if report.Outcome("a") != (hub.InterruptedOutcome{}) || report.Outcome("b") != (hub.InterruptedOutcome{}) {
		t.Errorf("unexpected robot outcomes: %+v", report)
	}
}
