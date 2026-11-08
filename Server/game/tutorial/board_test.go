package tutorial_test

import (
	json "encoding/json/v2"
	"fmt"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	"github.com/dvpico/Programming-Learning-AR/Server/game/tutorial"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
)

func TestTutorialClearConditions(t *testing.T) {
	t.Parallel()

	const goal = `{"type":"goal","position":{"x":2,"y":0}}`
	const checkpoints = `{"type":"checkpoint","position":{"x":1,"y":0},"order":20},
		{"type":"checkpoint","position":{"x":2,"y":0},"order":10}`

	tests := []struct {
		name      string
		condition string
		objects   string
		actions   []actions.Action
		want      boardmsg.TutorialResult
	}{
		{
			name:      "reach goal",
			condition: schema.ReachGoalType,
			objects:   goal,
			actions:   []actions.Action{actions.ForwardAction{}, actions.ForwardAction{}},
			want:      boardmsg.NewTutorialResult(2, boardmsg.EndConditionMet),
		},
		{
			name:      "stop before goal",
			condition: schema.ReachGoalType,
			objects:   goal,
			actions:   []actions.Action{actions.ForwardAction{}},
			want:      boardmsg.NewTutorialResult(1, boardmsg.EndAllStopped),
		},
		{
			name:      "revisit a checkpoint in order",
			condition: schema.VisitCheckpointsType,
			objects:   checkpoints,
			actions:   []actions.Action{actions.ForwardAction{}, actions.ForwardAction{}, actions.BackAction{}},
			want:      boardmsg.NewTutorialResult(3, boardmsg.EndConditionMet),
		},
		{
			name:      "visit checkpoints in the wrong order",
			condition: schema.VisitCheckpointsType,
			objects:   checkpoints,
			actions:   []actions.Action{actions.ForwardAction{}, actions.ForwardAction{}},
			want:      boardmsg.NewTutorialResult(2, boardmsg.EndAllStopped),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stageJSON := fmt.Sprintf(`{
				"variants":[{
					"id":"field-1",
					"width":4,"height":1,
					"terrain":[["land","land","land","sea"]],
					"objects":[%s],
					"robot":{"position":{"x":0,"y":0},"direction":"east"},
					"drillLimit":0
				}],
				"clearCondition":{"type":%q}
			}`, tt.objects, tt.condition)
			var stage schema.StageDefinition
			if err := json.Unmarshal([]byte(stageJSON), &stage); err != nil {
				t.Fatal(err)
			}
			const id = "p1"
			b, err := tutorial.NewBoard(stage, 0, id, 10)
			if err != nil {
				t.Fatal(err)
			}
			r := &tutorialRobot{actions: tt.actions}
			h, err := hub.New(b, []hub.Participant{{ID: id, Robot: r}})
			if err != nil {
				t.Fatal(err)
			}

			report, runErr := h.Run(hub.Options{MaxResumes: 10})

			if runErr != nil || report.Result() != tt.want {
				t.Errorf("result = %+v, %v, want %+v", report.Result(), runErr, tt.want)
			}
			var wantOutcome hub.Outcome = hub.CompletedOutcome{}
			if tt.want.Cleared() {
				wantOutcome = hub.InterruptedOutcome{}
			}
			if got := report.Outcome(id); got != wantOutcome {
				t.Errorf("robot outcome = %T, want %T", got, wantOutcome)
			}
		})
	}
}

type tutorialRobot struct{ actions []actions.Action }

func (r *tutorialRobot) Start() (robotmsg.RobotOutput, error) {
	if len(r.actions) == 0 {
		return robotmsg.RobotOutput{Step: robotmsg.DoneStep{}}, nil
	}
	action := r.actions[0]
	r.actions = r.actions[1:]
	return robotmsg.RobotOutput{Step: robotmsg.ActionStep{Action: action}}, nil
}

func (r *tutorialRobot) Next(values.Value) (robotmsg.RobotOutput, error) {
	return r.Start()
}
