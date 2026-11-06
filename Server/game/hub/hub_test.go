package hub_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game"
	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

type robotFunc func(values.Value) (robotmsg.Step, error)

func (f robotFunc) Start() (robotmsg.RobotOutput, error) {
	step, err := f(nil)
	return robotmsg.RobotOutput{Step: step}, err
}

func (f robotFunc) Next(value values.Value) (robotmsg.RobotOutput, error) {
	step, err := f(value)
	return robotmsg.RobotOutput{Step: step}, err
}

type robotFuncs struct {
	startEvents, nextEvents []robotevent.PicolangEvent
	start                   func() (robotmsg.Step, error)
	next                    func(values.Value) (robotmsg.Step, error)
}

func (r robotFuncs) Start() (robotmsg.RobotOutput, error) {
	step, err := r.start()
	return robotmsg.RobotOutput{Step: step, Events: r.startEvents}, err
}

func (r robotFuncs) Next(value values.Value) (robotmsg.RobotOutput, error) {
	step, err := r.next(value)
	return robotmsg.RobotOutput{Step: step, Events: r.nextEvents}, err
}

type boardFunc struct {
	startEvents, nextEvents []fieldevent.FieldEvent
	start                   func([]protocol.RobotID) error
	next                    func(robotmsg.Reply) (boardmsg.Decision, error)
}

func (b boardFunc) Start(ids []protocol.RobotID) ([]fieldevent.FieldEvent, error) {
	if b.start != nil {
		return b.startEvents, b.start(ids)
	}
	return b.startEvents, nil
}

func (b boardFunc) Next(reply robotmsg.Reply) (boardmsg.BoardOutput, error) {
	decision, err := b.next(reply)
	return boardmsg.BoardOutput{Decision: decision, Events: b.nextEvents}, err
}

func limits() hub.Options { return hub.Options{MaxResumes: 100} }

func doneRobot() game.Robot {
	return robotFunc(func(values.Value) (robotmsg.Step, error) { return robotmsg.DoneStep{}, nil })
}

func newHub(t *testing.T, b game.Board, participants ...hub.Participant) *hub.Hub {
	t.Helper()
	h, err := hub.New(b, participants)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func grid(t *testing.T) *board.Board {
	t.Helper()
	b, err := board.New(board.Config{
		Terrain: [][]values.TerrainValue{{values.LandTerrain{Kind: values.PlainLand{}}, values.LandTerrain{Kind: values.PlainLand{}}, values.LandTerrain{Kind: values.PlainLand{}}}},
		Robots: map[protocol.RobotID]board.RobotState{
			"a": {Position: board.Position{X: 0, Y: 0}, Direction: board.East, DrillDurability: 3},
			"b": {Position: board.Position{X: 2, Y: 0}, Direction: board.West, DrillDurability: 3},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBoardOwnsOrderAndCompletion(t *testing.T) {
	t.Parallel()
	valueB := values.BoolValue{Bool: true}
	valueA := values.UnitValue{}
	events := []boardmsg.Decision{
		boardmsg.StartDecision{RobotID: "b"},
		boardmsg.StartDecision{RobotID: "a"},
		boardmsg.ResumeDecision{RobotID: "b", ReturnValue: valueB},
		boardmsg.ResumeDecision{RobotID: "a", ReturnValue: valueA},
		boardmsg.DoneDecision{},
	}
	index := 0
	var calls []protocol.RobotID
	makeRobot := func(id protocol.RobotID, action actions.Action, want values.Value) game.Robot {
		first := true
		return robotFunc(func(value values.Value) (robotmsg.Step, error) {
			calls = append(calls, id)
			if first {
				first = false
				if value != nil {
					t.Error("initial robot input must be nil")
				}
				return robotmsg.ActionStep{Action: action}, nil
			}
			if value == nil || value != want {
				t.Errorf("robot %s got %+v, want %+v", id, value, want)
			}
			return robotmsg.DoneStep{}, nil
		})
	}
	b := boardFunc{
		start: func(ids []protocol.RobotID) error {
			if !reflect.DeepEqual(ids, []protocol.RobotID{"a", "b"}) {
				t.Errorf("registration order changed: %v", ids)
			}
			return nil
		},
		next: func(reply robotmsg.Reply) (boardmsg.Decision, error) {
			if index >= len(events) {
				t.Error("unexpected board call")
				return nil, errors.New("no more board events")
			}
			var want robotmsg.Reply
			switch index {
			case 1:
				want = robotmsg.ActionReply{RobotID: "b", Action: actions.ForwardAction{}}
			case 2:
				want = robotmsg.ActionReply{RobotID: "a", Action: actions.RotateAction{Rotation: values.Left}}
			case 3:
				want = robotmsg.DoneReply{RobotID: "b"}
			case 4:
				want = robotmsg.DoneReply{RobotID: "a"}
			}
			if reply != want {
				t.Errorf("got reply %+v, want %+v", reply, want)
			}
			d := events[index]
			index++
			return d, nil
		},
	}
	h := newHub(t, b,
		hub.Participant{ID: "a", Robot: makeRobot("a", actions.RotateAction{Rotation: values.Left}, valueA)},
		hub.Participant{ID: "b", Robot: makeRobot("b", actions.ForwardAction{}, valueB)},
	)
	options := limits()
	report, err := h.Run(options)
	if err != nil {
		t.Error(err)
	}
	if report.Resumes != 4 || !reflect.DeepEqual(calls, []protocol.RobotID{"b", "a", "b", "a"}) {
		t.Errorf("unexpected run: %+v, calls %v", report, calls)
	}
	if _, err := h.Run(options); err == nil {
		t.Error("hub reused")
	}
}

func TestRobotFailureDoesNotStopOthers(t *testing.T) {
	t.Parallel()
	failure := errors.New("robot failed")
	b := grid(t)
	failed := robotFunc(func(values.Value) (robotmsg.Step, error) { return nil, failure })
	first := true
	other := robotFunc(func(value values.Value) (robotmsg.Step, error) {
		if first {
			first = false
			return robotmsg.ActionStep{Action: actions.ForwardAction{}}, nil
		}
		if value != (values.BoolValue{Bool: true}) {
			t.Errorf("other robot got wrong result: %+v", value)
		}
		return robotmsg.DoneStep{}, nil
	})
	h := newHub(t, b, hub.Participant{ID: "a", Robot: failed}, hub.Participant{ID: "b", Robot: other})
	report, err := h.Run(limits())
	failedOutcome, failedOK := report.Outcomes["a"].(hub.FailedOutcome)
	if err != nil || !failedOK || !errors.Is(failedOutcome.Err, failure) || report.Outcomes["b"] != (hub.CompletedOutcome{}) {
		t.Errorf("got %+v, %v", report, err)
	}
	if b.Snapshot().Robots["a"].Position.X != 0 || b.Snapshot().Robots["b"].Position.X != 1 {
		t.Error("unexpected final positions")
	}
}

func TestLimits(t *testing.T) {
	t.Parallel()
	t.Run("zero-cost action loop", func(t *testing.T) {
		t.Parallel()
		b := grid(t)
		query := robotFunc(func(values.Value) (robotmsg.Step, error) {
			return robotmsg.ActionStep{Action: actions.DrillDurabilityAction{}}, nil
		})
		h := newHub(t, b, hub.Participant{ID: "a", Robot: query}, hub.Participant{ID: "b", Robot: doneRobot()})
		report, err := h.Run(hub.Options{MaxResumes: 3})
		if !errors.Is(err, hub.ErrResumeLimit) || report.Resumes != 3 || b.Snapshot().Tick != 0 {
			t.Errorf("got %+v, %v", report, err)
		}
	})
}

func TestProtocolViolations(t *testing.T) {
	t.Parallel()
	tests := map[string][]boardmsg.Decision{
		"empty":               {nil},
		"typed nil decision":  {(*boardmsg.StartDecision)(nil)},
		"unknown robot":       {boardmsg.StartDecision{RobotID: "unknown"}},
		"finished robot":      {boardmsg.StartDecision{RobotID: "a"}, boardmsg.StartDecision{RobotID: "a"}},
		"early completion":    {boardmsg.DoneDecision{}},
		"resume before start": {boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.UnitValue{}}},
	}
	for name, events := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			i := 0
			b := boardFunc{next: func(robotmsg.Reply) (boardmsg.Decision, error) {
				if i >= len(events) {
					t.Error("protocol violation was not rejected")
					return nil, errors.New("no more board events")
				}
				d := events[i]
				i++
				return d, nil
			}}
			h := newHub(t, b, hub.Participant{ID: "a", Robot: doneRobot()})
			if _, err := h.Run(limits()); !errors.Is(err, hub.ErrProtocol) {
				t.Errorf("got %v, want protocol error", err)
			}
		})
	}
}

func TestBoardErrors(t *testing.T) {
	t.Parallel()
	failure := errors.New("failure")
	for _, start := range []bool{true, false} {
		b := boardFunc{
			start: func([]protocol.RobotID) error {
				if start {
					return failure
				}
				return nil
			},
			next: func(robotmsg.Reply) (boardmsg.Decision, error) { return nil, failure },
		}
		h := newHub(t, b, hub.Participant{ID: "a", Robot: doneRobot()})
		if _, err := h.Run(limits()); !errors.Is(err, failure) {
			t.Errorf("board error lost: %v", err)
		}
	}
}

func TestNewValidationAndInvalidSteps(t *testing.T) {
	t.Parallel()
	b := grid(t)
	for _, participants := range [][]hub.Participant{
		nil, {{ID: "", Robot: doneRobot()}}, {{ID: "a"}},
		{{ID: "a", Robot: doneRobot()}, {ID: "a", Robot: doneRobot()}},
	} {
		if _, err := hub.New(b, participants); err == nil {
			t.Error("invalid participants accepted")
		}
	}
	if _, err := hub.New(nil, []hub.Participant{{ID: "a", Robot: doneRobot()}}); err == nil {
		t.Error("nil board accepted")
	}
	for _, step := range []robotmsg.Step{
		nil, (*robotmsg.DoneStep)(nil), robotmsg.ActionStep{},
		robotmsg.ActionStep{Action: (*actions.ForwardAction)(nil)},
	} {
		r := robotFunc(func(values.Value) (robotmsg.Step, error) { return step, nil })
		h := newHub(t, grid(t), hub.Participant{ID: "a", Robot: r}, hub.Participant{ID: "b", Robot: doneRobot()})
		report, err := h.Run(limits())
		failed, ok := report.Outcomes["a"].(hub.FailedOutcome)
		if err != nil || !ok || !errors.Is(failed.Err, hub.ErrProtocol) {
			t.Errorf("invalid step did not isolate robot: %+v, %v", report, err)
		}
	}
	h := newHub(t, grid(t), hub.Participant{ID: "a", Robot: doneRobot()}, hub.Participant{ID: "b", Robot: doneRobot()})
	if _, err := h.Run(hub.Options{}); err == nil {
		t.Error("zero limit accepted")
	}
}

func TestRoutesAllActionBuiltins(t *testing.T) {
	t.Parallel()
	terrain := values.LandTerrain{Kind: values.GoalLand{}}
	boolean := values.BoolValue{Bool: true}
	integer := values.IntegerValue{Integer: 3}
	unit := values.UnitValue{}
	observation := terrain
	tests := []struct {
		action actions.Action
		result values.Value
	}{
		{action: actions.ForwardAction{}, result: boolean},
		{action: actions.BackAction{}, result: boolean},
		{action: actions.RotateAction{Rotation: values.Left}, result: unit},
		{action: actions.WaitAction{Integer: 7}, result: unit},
		{action: actions.PickUpAction{}, result: boolean},
		{action: actions.ReleaseAction{}, result: boolean},
		{action: actions.DrillAction{}, result: boolean},
		{action: actions.DrillDurabilityAction{}, result: integer},
		{action: actions.DistanceAction{}, result: integer},
		{action: actions.SearchFrontAction{}, result: observation},
		{action: actions.SearchNowAction{}, result: observation},
		{action: actions.PrintIntegerAction{Integer: -42}, result: unit},
		{action: actions.PrintBoolAction{Bool: true}, result: unit},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%T", tt.action), func(t *testing.T) {
			t.Parallel()
			boardCalls, robotCalls := 0, 0
			b := boardFunc{next: func(reply robotmsg.Reply) (boardmsg.Decision, error) {
				boardCalls++
				switch boardCalls {
				case 1:
					if reply != nil {
						t.Error("initial reply must be nil")
					}
					return boardmsg.StartDecision{RobotID: "a"}, nil
				case 2:
					if reply != (robotmsg.ActionReply{RobotID: "a", Action: tt.action}) {
						t.Errorf("action or arguments lost in transit: %+v", reply)
					}
					return boardmsg.ResumeDecision{RobotID: "a", ReturnValue: tt.result}, nil
				case 3:
					if reply != (robotmsg.DoneReply{RobotID: "a"}) {
						t.Errorf("completion lost in transit: %+v", reply)
					}
					return boardmsg.DoneDecision{}, nil
				default:
					t.Error("unexpected board call")
					return nil, nil
				}
			}}
			r := robotFunc(func(value values.Value) (robotmsg.Step, error) {
				robotCalls++
				if robotCalls == 1 {
					if value != nil {
						t.Error("initial return value must be nil")
					}
					return robotmsg.ActionStep{Action: tt.action}, nil
				}
				if value == nil || value != tt.result {
					t.Errorf("return value lost in transit: got %+v, want %+v", value, tt.result)
				}
				return robotmsg.DoneStep{}, nil
			})
			h := newHub(t, b, hub.Participant{ID: "a", Robot: r})
			report, err := h.Run(limits())
			if err != nil || report.Outcomes["a"] != (hub.CompletedOutcome{}) || report.Resumes != 2 || boardCalls != 3 || robotCalls != 2 {
				t.Errorf("unexpected result: %+v, %v", report, err)
			}
		})
	}
}

func TestTerrainObservationSurvivesBoardChanges(t *testing.T) {
	t.Parallel()
	var cell values.TerrainValue = values.LandTerrain{Kind: values.LuggageLand{}}
	observation := cell
	boardCalls, robotCalls := 0, 0
	b := boardFunc{next: func(reply robotmsg.Reply) (boardmsg.Decision, error) {
		boardCalls++
		switch boardCalls {
		case 1:
			return boardmsg.StartDecision{RobotID: "a"}, nil
		case 2:
			if reply != (robotmsg.ActionReply{RobotID: "a", Action: actions.SearchFrontAction{}}) {
				t.Error("expected search_front")
			}
			// Copy the observation before the live cell changes.
			result := cell
			cell = values.SeaTerrain{}
			return boardmsg.ResumeDecision{RobotID: "a", ReturnValue: result}, nil
		case 3:
			if reply != (robotmsg.DoneReply{RobotID: "a"}) {
				t.Errorf("expected completion after local terrain predicate, got %+v", reply)
			}
			return boardmsg.DoneDecision{}, nil
		default:
			t.Error("unexpected board call")
			return nil, errors.New("unexpected board call")
		}
	}}
	r := robotFunc(func(value values.Value) (robotmsg.Step, error) {
		robotCalls++
		switch robotCalls {
		case 1:
			return robotmsg.ActionStep{Action: actions.SearchFrontAction{}}, nil
		case 2:
			if value == nil {
				t.Error("expected terrain return value")
				return nil, errors.New("missing terrain return value")
			}
			terrain, ok := value.(values.TerrainValue)
			if !ok {
				t.Error("expected terrain return value")
				return nil, errors.New("unexpected terrain return value")
			}
			if terrain != observation {
				t.Errorf("terrain observation changed in transit: %+v", terrain)
			}
			// Terrain predicates are evaluated inside the robot without an action.
			if _, isLand := terrain.(values.LandTerrain); !isLand {
				t.Error("expected local is_land result to be true")
			}
			return robotmsg.DoneStep{}, nil
		default:
			t.Error("unexpected robot call")
			return nil, errors.New("unexpected robot call")
		}
	})
	h := newHub(t, b, hub.Participant{ID: "a", Robot: r})
	report, err := h.Run(limits())
	if err != nil {
		t.Error(err)
	}
	if report.Outcomes["a"] != (hub.CompletedOutcome{}) || report.Resumes != 2 || boardCalls != 3 || robotCalls != 2 {
		t.Errorf("local terrain predicate caused unexpected execution: %+v, board calls=%d, robot calls=%d", report, boardCalls, robotCalls)
	}
	if cell != (values.SeaTerrain{}) {
		t.Errorf("live cell did not change: %+v", cell)
	}
}

func TestStartsOnceThenResumesWithValue(t *testing.T) {
	t.Parallel()
	for _, value := range []values.Value{
		values.BoolValue{Bool: false}, values.IntegerValue{Integer: -42}, values.UnitValue{},
		values.LandTerrain{Kind: values.PlainLand{}}, values.LandTerrain{Kind: values.GoalLand{}}, values.LandTerrain{Kind: values.CheckpointLand{Order: 2}}, values.LandTerrain{Kind: values.LuggageLand{}}, values.RockTerrain{}, values.SeaTerrain{},
	} {
		t.Run(fmt.Sprintf("%T", value), func(t *testing.T) {
			t.Parallel()
			starts, resumes, boardCalls := 0, 0, 0
			r := robotFuncs{
				start: func() (robotmsg.Step, error) {
					starts++
					return robotmsg.ActionStep{Action: actions.SearchNowAction{}}, nil
				},
				next: func(got values.Value) (robotmsg.Step, error) {
					resumes++
					if starts != 1 || got != value {
						t.Errorf("resumed after %d starts with %+v, want %+v", starts, got, value)
					}
					return robotmsg.DoneStep{}, nil
				},
			}
			b := boardFunc{next: func(reply robotmsg.Reply) (boardmsg.Decision, error) {
				boardCalls++
				switch boardCalls {
				case 1:
					if reply != nil {
						t.Errorf("unexpected initial reply: %+v", reply)
					}
					return boardmsg.StartDecision{RobotID: "a"}, nil
				case 2:
					if reply != (robotmsg.ActionReply{RobotID: "a", Action: actions.SearchNowAction{}}) {
						t.Errorf("unexpected action reply: %+v", reply)
					}
					return boardmsg.ResumeDecision{RobotID: "a", ReturnValue: value}, nil
				case 3:
					if reply != (robotmsg.DoneReply{RobotID: "a"}) {
						t.Errorf("unexpected completion reply: %+v", reply)
					}
					return boardmsg.DoneDecision{}, nil
				default:
					t.Error("unexpected board call")
					return nil, errors.New("no more board events")
				}
			}}
			report, err := newHub(t, b, hub.Participant{ID: "a", Robot: r}).Run(limits())
			if err != nil || starts != 1 || resumes != 1 || report.Resumes != 2 || report.Outcomes["a"] != (hub.CompletedOutcome{}) {
				t.Errorf("got %+v, %v with %d starts and %d resumes", report, err, starts, resumes)
			}
		})
	}
}

func TestRejectsInvalidStartAndResume(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		events  []boardmsg.Decision
		started uint64
	}{
		{
			name:   "resume before start",
			events: []boardmsg.Decision{boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.UnitValue{}}},
		},
		{
			name:    "duplicate start",
			events:  []boardmsg.Decision{boardmsg.StartDecision{RobotID: "a"}, boardmsg.StartDecision{RobotID: "a"}},
			started: 1,
		},
		{
			name:    "missing return value",
			events:  []boardmsg.Decision{boardmsg.StartDecision{RobotID: "a"}, boardmsg.ResumeDecision{RobotID: "a"}},
			started: 1,
		},
		{
			name:    "typed nil return value",
			events:  []boardmsg.Decision{boardmsg.StartDecision{RobotID: "a"}, boardmsg.ResumeDecision{RobotID: "a", ReturnValue: (*values.BoolValue)(nil)}},
			started: 1,
		},
		{
			name:    "typed nil terrain",
			events:  []boardmsg.Decision{boardmsg.StartDecision{RobotID: "a"}, boardmsg.ResumeDecision{RobotID: "a", ReturnValue: (*values.LandTerrain)(nil)}},
			started: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var starts, resumes uint64
			index := 0
			r := robotFuncs{
				start: func() (robotmsg.Step, error) {
					starts++
					return robotmsg.ActionStep{Action: actions.DrillDurabilityAction{}}, nil
				},
				next: func(values.Value) (robotmsg.Step, error) {
					resumes++
					return robotmsg.DoneStep{}, nil
				},
			}
			b := boardFunc{next: func(robotmsg.Reply) (boardmsg.Decision, error) {
				if index >= len(tt.events) {
					t.Error("invalid request was not rejected")
					return nil, errors.New("no more board events")
				}
				decision := tt.events[index]
				index++
				return decision, nil
			}}
			report, err := newHub(t, b, hub.Participant{ID: "a", Robot: r}).Run(limits())
			if !errors.Is(err, hub.ErrProtocol) || starts != tt.started || resumes != 0 || report.Resumes != tt.started {
				t.Errorf("got %+v, %v with %d starts and %d resumes", report, err, starts, resumes)
			}
		})
	}
}
