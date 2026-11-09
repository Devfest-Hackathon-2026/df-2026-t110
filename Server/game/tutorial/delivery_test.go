package tutorial_test

import (
	json "encoding/json/v2"
	"fmt"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	"github.com/dvpico/Programming-Learning-AR/Server/game/tutorial"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
)

func deliveryStage(t *testing.T, objects string, count int) schema.StageDefinition {
	t.Helper()
	var stage schema.StageDefinition
	data := fmt.Sprintf(`{
		"variants":[{"id":"delivery","width":7,"height":2,
		"terrain":[["land","land","land","land","land","land","sea"],
		           ["land","land","land","land","land","land","sea"]],
		"objects":[%s],"robot":{"position":{"x":0,"y":0},"direction":"east"},"drillLimit":0}],
		"clearCondition":{"type":"deliver_luggage","requiredCount":%d}
	}`, objects, count)
	if err := json.Unmarshal([]byte(data), &stage); err != nil {
		t.Fatal(err)
	}
	return stage
}

func runDelivery(t *testing.T, stage schema.StageDefinition, program []actions.Action, tick uint64, cleared bool) *board.Board {
	t.Helper()
	b, err := tutorial.NewBoard(stage, 0, "p1", 100)
	if err != nil {
		t.Fatal(err)
	}
	h, err := hub.New(b, []hub.Participant{{ID: "p1", Robot: &tutorialRobot{actions: program}}})
	if err != nil {
		t.Fatal(err)
	}
	report, err := h.Run(hub.Options{MaxResumes: 100})
	if err != nil {
		t.Fatal(err)
	}
	reason := boardmsg.EndAllStopped
	if cleared {
		reason = boardmsg.EndConditionMet
	}
	if want := boardmsg.NewTutorialResult(tick, reason); report.Result() != want {
		t.Errorf("result = %+v, want %+v", report.Result(), want)
	}
	return b
}

func TestDeliverLuggage(t *testing.T) {
	t.Parallel()
	const objects = `
		{"type":"luggage","id":"parcel","position":{"x":1,"y":0}},
		{"type":"territory","position":{"x":3,"y":0}},
		{"type":"goal","position":{"x":5,"y":0}}`
	forward := actions.ForwardAction{}
	pick := actions.PickUpAction{}
	release := actions.ReleaseAction{}
	for _, tt := range []struct {
		name    string
		program []actions.Action
		cleared bool
	}{
		{name: "robot alone reaches territory", program: []actions.Action{forward, forward, forward}},
		{name: "robot alone reaches goal", program: []actions.Action{forward, forward, forward, forward, forward}},
		{name: "carried luggage on territory", program: []actions.Action{pick, forward, forward, forward}},
		{name: "release on plain land", program: []actions.Action{pick, forward, release}},
		{name: "release away from territory", program: []actions.Action{pick, forward, forward, forward, release}},
		{name: "discard into sea", program: []actions.Action{pick, forward, forward, forward, forward, forward, release}},
		{name: "release without luggage", program: []actions.Action{forward, forward, release}},
		{name: "deliver to territory", program: []actions.Action{pick, forward, forward, release}, cleared: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stage := deliveryStage(t, objects, 1)
			program := append([]actions.Action{}, tt.program...)
			if tt.cleared {
				// Clearing must interrupt the program on the release turn.
				program = append(program, actions.WaitAction{Integer: 50})
			}
			b := runDelivery(t, stage, program, uint64(len(tt.program)), tt.cleared)
			if tt.cleared {
				robot := b.Snapshot().Robots["p1"]
				if robot.CarryingLuggage() || robot.TeamID() != board.NoTeam {
					t.Errorf("robot state after delivery = %+v", robot)
				}
				cell := b.Snapshot().Terrain[robot.Position().Y()][robot.Position().X()].(board.LandTerrain)
				if cell.TerritoryOwner() != board.NoTeam {
					t.Error("test must clear while the robot is outside its destination")
				}
			}
		})
	}
}

func TestDeliverThreeLuggageToThreeCells(t *testing.T) {
	t.Parallel()
	const objects = `
		{"type":"luggage","id":"a","position":{"x":1,"y":0}},
		{"type":"luggage","id":"b","position":{"x":3,"y":0}},
		{"type":"luggage","id":"c","position":{"x":5,"y":0}},
		{"type":"territory","position":{"x":1,"y":1}},
		{"type":"territory","position":{"x":3,"y":1}},
		{"type":"territory","position":{"x":5,"y":1}}`
	// Pick up the next parcel, step onto its old cell, and place it to the south.
	trip := []actions.Action{actions.PickUpAction{}, actions.ForwardAction{}, actions.RotateAction{Rotation: values.Right}, actions.ReleaseAction{}}
	program := append([]actions.Action{}, trip...)
	program = append(program, actions.RotateAction{Rotation: values.Left}, actions.ForwardAction{})
	program = append(program, trip...)
	for _, tt := range []struct {
		name    string
		count   int
		all     bool
		cleared bool
	}{
		{name: "two deliveries are insufficient", count: 3},
		{name: "three deliveries clear", count: 3, all: true, cleared: true},
		{name: "required count can be below capacity", count: 2, cleared: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			steps := append([]actions.Action{}, program...)
			if tt.all {
				steps = append(steps, actions.RotateAction{Rotation: values.Left}, actions.ForwardAction{})
				steps = append(steps, trip...)
			}
			b := runDelivery(t, deliveryStage(t, objects, tt.count), steps, uint64(len(steps)), tt.cleared)
			for _, x := range []int{1, 3, 5} {
				land := b.Snapshot().Terrain[1][x].(board.LandTerrain)
				if want := x != 5 || tt.all; land.HasLuggage() != want {
					t.Errorf("luggage at (%d, 1) = %v, want %v", x, land.HasLuggage(), want)
				}
			}
		})
	}
}

func TestReachGoalDoesNotRequireDelivery(t *testing.T) {
	t.Parallel()
	stage := deliveryStage(t, `
		{"type":"luggage","id":"parcel","position":{"x":1,"y":0}},
		{"type":"goal","position":{"x":2,"y":0}}`, 1)
	stage.ClearCondition = schema.ClearCondition{ReachGoal: &schema.ReachGoal{Type: schema.ReachGoalType}}
	b := runDelivery(t, stage, []actions.Action{actions.ForwardAction{}, actions.ForwardAction{}}, 2, true)
	if !b.Snapshot().Terrain[0][1].(board.LandTerrain).HasLuggage() {
		t.Error("reach_goal must clear with the luggage left at its starting cell")
	}
}

func TestDeliveryBoardValidation(t *testing.T) {
	t.Parallel()
	const objects = `{"type":"luggage","id":"a","position":{"x":1,"y":0}},
		{"type":"territory","position":{"x":3,"y":0}}`
	for _, tt := range []struct {
		name   string
		change func(*schema.StageDefinition)
	}{
		{name: "zero count", change: func(s *schema.StageDefinition) { s.ClearCondition.DeliverLuggage.RequiredCount = 0 }},
		{name: "negative count", change: func(s *schema.StageDefinition) { s.ClearCondition.DeliverLuggage.RequiredCount = -1 }},
		{name: "missing destination", change: func(s *schema.StageDefinition) { s.Variants[0].Objects = s.Variants[0].Objects[:1] }},
		{name: "insufficient destination cells", change: func(s *schema.StageDefinition) {
			s.ClearCondition.DeliverLuggage.RequiredCount = 2
			s.Variants[0].Objects = append(s.Variants[0].Objects, schema.FieldObject{Luggage: &schema.Luggage{
				Type: schema.LuggageType, Id: "b", Position: schema.Position{X: 4, Y: 0},
			}})
		}},
		{name: "insufficient luggage", change: func(s *schema.StageDefinition) {
			s.ClearCondition.DeliverLuggage.RequiredCount = 2
			s.Variants[0].Objects = append(s.Variants[0].Objects, schema.FieldObject{Territory: &schema.Territory{
				Type: schema.TerritoryType, Position: schema.Position{X: 4, Y: 0},
			}})
		}},
		{name: "missing luggage", change: func(s *schema.StageDefinition) { s.Variants[0].Objects = s.Variants[0].Objects[1:] }},
		{name: "multiple conditions", change: func(s *schema.StageDefinition) {
			s.ClearCondition.ReachGoal = &schema.ReachGoal{Type: schema.ReachGoalType}
		}},
		{name: "multiple object variants", change: func(s *schema.StageDefinition) { s.Variants[0].Objects[1].Goal = &schema.Goal{} }},
		{name: "territory outside field", change: func(s *schema.StageDefinition) { s.Variants[0].Objects[1].Territory.Position.X = 7 }},
		{name: "territory on sea", change: func(s *schema.StageDefinition) { s.Variants[0].Objects[1].Territory.Position.X = 6 }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stage := deliveryStage(t, objects, 1)
			tt.change(&stage)
			if _, err := tutorial.NewBoard(stage, 0, "p1", 100); err == nil {
				t.Error("invalid delivery board accepted")
			}
		})
	}
}

func TestTerritoryObjectOverlaps(t *testing.T) {
	t.Parallel()
	const territory = `{"type":"territory","position":{"x":1,"y":0}}`
	for _, tt := range []struct {
		name   string
		object string
		valid  bool
	}{
		{name: "luggage", object: `{"type":"luggage","id":"a","position":{"x":1,"y":0}}`, valid: true},
		{name: "goal", object: `{"type":"goal","position":{"x":1,"y":0}}`},
		{name: "checkpoint", object: `{"type":"checkpoint","order":1,"position":{"x":1,"y":0}}`},
		{name: "duplicate territory", object: territory},
	} {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/reverse=%v", tt.name, reverse), func(t *testing.T) {
				t.Parallel()
				objects := territory + "," + tt.object
				if reverse {
					objects = tt.object + "," + territory
				}
				stage := deliveryStage(t, objects, 1)
				b, err := tutorial.NewBoard(stage, 0, "p1", 100)
				if (err == nil) != tt.valid {
					t.Errorf("NewBoard error = %v, valid = %v", err, tt.valid)
				}
				if err == nil && tt.valid {
					land := b.Snapshot().Terrain[0][1].(board.LandTerrain)
					if !land.HasLuggage() || land.TerritoryOwner() == board.NoTeam {
						t.Errorf("overlapping luggage and territory = %+v", land)
					}
				}
			})
		}
	}
}
