package hub_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game"
	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
)

type robotFunc func(context.Context, game.Value) (game.Step, error)

func (f robotFunc) Start(ctx context.Context) (game.Step, error) {
	return f(ctx, nil)
}

func (f robotFunc) Next(ctx context.Context, value game.Value) (game.Step, error) {
	return f(ctx, value)
}

type robotFuncs struct {
	start func(context.Context) (game.Step, error)
	next  func(context.Context, game.Value) (game.Step, error)
}

func (r robotFuncs) Start(ctx context.Context) (game.Step, error) {
	return r.start(ctx)
}

func (r robotFuncs) Next(ctx context.Context, value game.Value) (game.Step, error) {
	return r.next(ctx, value)
}

type boardFunc struct {
	start func(context.Context, []game.RobotID) error
	next  func(context.Context, game.Reply) (game.Decision, error)
}

func (b boardFunc) Start(ctx context.Context, ids []game.RobotID) error {
	if b.start != nil {
		return b.start(ctx, ids)
	}
	return nil
}

func (b boardFunc) Next(ctx context.Context, reply game.Reply) (game.Decision, error) {
	return b.next(ctx, reply)
}

func limits() hub.Options { return hub.Options{MaxTicks: 10, MaxResumes: 100} }

func doneRobot() game.Robot {
	return robotFunc(func(context.Context, game.Value) (game.Step, error) { return game.DoneStep{}, nil })
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
		Terrain: [][]game.TerrainValue{{game.LandTerrain{Kind: game.PlainLand{}}, game.LandTerrain{Kind: game.PlainLand{}}, game.LandTerrain{Kind: game.PlainLand{}}}},
		Robots: map[game.RobotID]board.RobotState{
			"a": {Position: board.Position{X: 0, Y: 0}, Direction: board.East, DrillDurability: 3},
			"b": {Position: board.Position{X: 2, Y: 0}, Direction: board.West, DrillDurability: 3},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBoardOwnsOrderAndMultiTickCompletion(t *testing.T) {
	t.Parallel()
	// This substitute board finishes b's action at tick 2 and a's at tick 5.
	// The hub has no knowledge of action durations or scheduling rules.
	two, five := uint64(2), uint64(5)
	valueB := game.BoolValue{Bool: true}
	valueA := game.UnitValue{}
	events := []game.Decision{
		game.StartDecision{RobotID: "b"},
		game.StartDecision{RobotID: "a"},
		game.AdvanceDecision{Tick: two},
		game.UpdateDecision{Update: game.TickUpdate{Tick: 2}},
		game.ResumeDecision{RobotID: "b", ReturnValue: valueB},
		game.AdvanceDecision{Tick: five},
		game.UpdateDecision{Update: game.TickUpdate{Tick: 5}},
		game.ResumeDecision{RobotID: "a", ReturnValue: valueA},
		game.DoneDecision{},
	}
	index := 0
	var calls []game.RobotID
	makeRobot := func(id game.RobotID, action game.Action, want game.Value) game.Robot {
		first := true
		return robotFunc(func(_ context.Context, value game.Value) (game.Step, error) {
			calls = append(calls, id)
			if first {
				first = false
				if value != nil {
					t.Error("initial robot input must be nil")
				}
				return game.ActionStep{Action: action}, nil
			}
			if value == nil || value != want {
				t.Errorf("robot %s got %+v, want %+v", id, value, want)
			}
			return game.DoneStep{}, nil
		})
	}
	b := boardFunc{
		start: func(_ context.Context, ids []game.RobotID) error {
			if !reflect.DeepEqual(ids, []game.RobotID{"a", "b"}) {
				t.Errorf("registration order changed: %v", ids)
			}
			return nil
		},
		next: func(_ context.Context, reply game.Reply) (game.Decision, error) {
			if index >= len(events) {
				t.Error("unexpected board call")
				return nil, errors.New("no more board events")
			}
			var want game.Reply
			switch index {
			case 1:
				want = game.ActionReply{RobotID: "b", Action: game.ForwardAction{}}
			case 2:
				want = game.ActionReply{RobotID: "a", Action: game.RotateAction{Rotation: game.Left}}
			case 5:
				want = game.DoneReply{RobotID: "b"}
			case 8:
				want = game.DoneReply{RobotID: "a"}
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
		hub.Participant{ID: "a", Robot: makeRobot("a", game.RotateAction{Rotation: game.Left}, valueA)},
		hub.Participant{ID: "b", Robot: makeRobot("b", game.ForwardAction{}, valueB)},
	)
	var ticks []uint64
	options := limits()
	options.OnTick = func(update game.TickUpdate) error { ticks = append(ticks, update.Tick); return nil }
	report, err := h.Run(t.Context(), options)
	if err != nil {
		t.Error(err)
	}
	if report.Tick != 5 || report.Resumes != 4 || !reflect.DeepEqual(calls, []game.RobotID{"b", "a", "b", "a"}) || !reflect.DeepEqual(ticks, []uint64{2, 5}) {
		t.Errorf("unexpected run: %+v, calls %v, ticks %v", report, calls, ticks)
	}
	if _, err := h.Run(t.Context(), options); err == nil {
		t.Error("hub reused")
	}
}

func TestRobotFailureDoesNotStopOthers(t *testing.T) {
	t.Parallel()
	failure := errors.New("robot failed")
	b := grid(t)
	failed := robotFunc(func(context.Context, game.Value) (game.Step, error) { return nil, failure })
	first := true
	other := robotFunc(func(_ context.Context, value game.Value) (game.Step, error) {
		if first {
			first = false
			return game.ActionStep{Action: game.ForwardAction{}}, nil
		}
		if value != (game.BoolValue{Bool: true}) {
			t.Errorf("other robot got wrong result: %+v", value)
		}
		if robots := b.Snapshot().Robots; len(robots) != 1 || robots["b"].Position.X != 1 {
			t.Errorf("unexpected robots after the first tick: %+v", robots)
		}
		return game.DoneStep{}, nil
	})
	h := newHub(t, b, hub.Participant{ID: "a", Robot: failed}, hub.Participant{ID: "b", Robot: other})
	report, err := h.Run(t.Context(), limits())
	failedOutcome, failedOK := report.Outcomes["a"].(hub.FailedOutcome)
	if err != nil || report.Tick != 1 || !failedOK || !errors.Is(failedOutcome.Err, failure) || report.Outcomes["b"] != (hub.CompletedOutcome{}) {
		t.Errorf("got %+v, %v", report, err)
	}
	if robots := b.Snapshot().Robots; len(robots) != 0 {
		t.Errorf("stopped robots remained on the board: %+v", robots)
	}
}

func TestLimits(t *testing.T) {
	t.Parallel()
	t.Run("zero-cost action loop", func(t *testing.T) {
		t.Parallel()
		b := grid(t)
		query := robotFunc(func(context.Context, game.Value) (game.Step, error) {
			return game.ActionStep{Action: game.DrillDurabilityAction{}}, nil
		})
		h := newHub(t, b, hub.Participant{ID: "a", Robot: query}, hub.Participant{ID: "b", Robot: doneRobot()})
		report, err := h.Run(t.Context(), hub.Options{MaxTicks: 10, MaxResumes: 3})
		if !errors.Is(err, hub.ErrResumeLimit) || report.Resumes != 3 || b.Snapshot().Tick != 0 {
			t.Errorf("got %+v, %v", report, err)
		}
	})
	t.Run("before atomic commit", func(t *testing.T) {
		t.Parallel()
		b := grid(t)
		move := robotFunc(func(context.Context, game.Value) (game.Step, error) {
			return game.ActionStep{Action: game.RotateAction{Rotation: game.Right}}, nil
		})
		h := newHub(t, b, hub.Participant{ID: "a", Robot: move}, hub.Participant{ID: "b", Robot: move})
		report, err := h.Run(t.Context(), hub.Options{MaxTicks: 1, MaxResumes: 100})
		if !errors.Is(err, hub.ErrTickLimit) || report.Tick != 1 || b.Snapshot().Tick != 1 || b.Snapshot().Robots["a"].Direction != board.South {
			t.Errorf("got %+v, %v, board %+v", report, err, b.Snapshot())
		}
	})
	t.Run("future tick leap rejected", func(t *testing.T) {
		t.Parallel()
		calls := 0
		five := uint64(5)
		b := boardFunc{next: func(context.Context, game.Reply) (game.Decision, error) {
			calls++
			return game.AdvanceDecision{Tick: five}, nil
		}}
		h := newHub(t, b, hub.Participant{ID: "a", Robot: doneRobot()})
		_, err := h.Run(t.Context(), hub.Options{MaxTicks: 4, MaxResumes: 10})
		if !errors.Is(err, hub.ErrTickLimit) || calls != 1 {
			t.Errorf("board called beyond limit: %d, %v", calls, err)
		}
	})
}

func TestProtocolViolations(t *testing.T) {
	t.Parallel()
	one, two := uint64(1), uint64(2)
	tests := map[string][]game.Decision{
		"empty":               {nil},
		"typed nil decision":  {(*game.StartDecision)(nil)},
		"unknown robot":       {game.StartDecision{RobotID: "unknown"}},
		"finished robot":      {game.StartDecision{RobotID: "a"}, game.StartDecision{RobotID: "a"}},
		"early completion":    {game.DoneDecision{}},
		"unannounced update":  {game.UpdateDecision{Update: game.TickUpdate{Tick: 1}}},
		"wrong commit tick":   {game.AdvanceDecision{Tick: one}, game.UpdateDecision{Update: game.TickUpdate{Tick: 2}}},
		"missing commit":      {game.AdvanceDecision{Tick: one}, game.StartDecision{RobotID: "a"}},
		"nonincreasing tick":  {game.AdvanceDecision{Tick: two}, game.UpdateDecision{Update: game.TickUpdate{Tick: 2}}, game.AdvanceDecision{Tick: one}},
		"resume before start": {game.ResumeDecision{RobotID: "a", ReturnValue: game.UnitValue{}}},
	}
	for name, events := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			i := 0
			b := boardFunc{next: func(context.Context, game.Reply) (game.Decision, error) {
				if i >= len(events) {
					t.Error("protocol violation was not rejected")
					return nil, errors.New("no more board events")
				}
				d := events[i]
				i++
				return d, nil
			}}
			h := newHub(t, b, hub.Participant{ID: "a", Robot: doneRobot()})
			if _, err := h.Run(t.Context(), limits()); !errors.Is(err, hub.ErrProtocol) {
				t.Errorf("got %v, want protocol error", err)
			}
		})
	}
}

func TestBoardAndCallbackErrors(t *testing.T) {
	t.Parallel()
	failure := errors.New("failure")
	for _, start := range []bool{true, false} {
		b := boardFunc{
			start: func(context.Context, []game.RobotID) error {
				if start {
					return failure
				}
				return nil
			},
			next: func(context.Context, game.Reply) (game.Decision, error) { return nil, failure },
		}
		h := newHub(t, b, hub.Participant{ID: "a", Robot: doneRobot()})
		if _, err := h.Run(t.Context(), limits()); !errors.Is(err, failure) {
			t.Errorf("board error lost: %v", err)
		}
	}
	b := grid(t)
	move := robotFunc(func(context.Context, game.Value) (game.Step, error) {
		return game.ActionStep{Action: game.ForwardAction{}}, nil
	})
	h := newHub(t, b, hub.Participant{ID: "a", Robot: move}, hub.Participant{ID: "b", Robot: move})
	options := limits()
	options.OnTick = func(game.TickUpdate) error { return failure }
	report, err := h.Run(t.Context(), options)
	if !errors.Is(err, failure) || report.Tick != 1 {
		t.Errorf("callback error lost: %+v, %v", report, err)
	}
}

func TestCancellationDiscardsPendingActions(t *testing.T) {
	t.Parallel()
	b := grid(t)
	ctx, cancel := context.WithCancel(t.Context())
	move := robotFunc(func(context.Context, game.Value) (game.Step, error) {
		return game.ActionStep{Action: game.ForwardAction{}}, nil
	})
	interrupted := robotFunc(func(context.Context, game.Value) (game.Step, error) {
		cancel()
		return nil, context.Canceled
	})
	h := newHub(t, b, hub.Participant{ID: "a", Robot: move}, hub.Participant{ID: "b", Robot: interrupted})
	report, err := h.Run(ctx, limits())
	if !errors.Is(err, context.Canceled) || report.Tick != 0 || b.Snapshot().Robots["a"].Position.X != 0 {
		t.Errorf("got %+v, %v", report, err)
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
	for _, step := range []game.Step{
		nil, (*game.DoneStep)(nil), game.ActionStep{},
		game.ActionStep{Action: (*game.ForwardAction)(nil)},
	} {
		r := robotFunc(func(context.Context, game.Value) (game.Step, error) { return step, nil })
		h := newHub(t, grid(t), hub.Participant{ID: "a", Robot: r}, hub.Participant{ID: "b", Robot: doneRobot()})
		report, err := h.Run(t.Context(), limits())
		failed, ok := report.Outcomes["a"].(hub.FailedOutcome)
		if err != nil || !ok || !errors.Is(failed.Err, hub.ErrProtocol) {
			t.Errorf("invalid step did not isolate robot: %+v, %v", report, err)
		}
	}
	for _, options := range []hub.Options{{}, {MaxTicks: 1}, {MaxResumes: 1}} {
		h := newHub(t, grid(t), hub.Participant{ID: "a", Robot: doneRobot()}, hub.Participant{ID: "b", Robot: doneRobot()})
		if _, err := h.Run(t.Context(), options); err == nil {
			t.Error("zero limit accepted")
		}
	}
}

func TestRoutesAllActionBuiltins(t *testing.T) {
	t.Parallel()
	terrain := game.LandTerrain{Kind: game.GoalLand{}}
	boolean := game.BoolValue{Bool: true}
	integer := game.IntegerValue{Integer: 3}
	unit := game.UnitValue{}
	observation := terrain
	tests := []struct {
		action game.Action
		result game.Value
	}{
		{action: game.ForwardAction{}, result: boolean},
		{action: game.BackAction{}, result: boolean},
		{action: game.RotateAction{Rotation: game.Left}, result: unit},
		{action: game.WaitAction{Integer: 7}, result: unit},
		{action: game.PickUpAction{}, result: boolean},
		{action: game.ReleaseAction{}, result: boolean},
		{action: game.DrillAction{}, result: boolean},
		{action: game.DrillDurabilityAction{}, result: integer},
		{action: game.DistanceAction{Target: game.DistanceRock}, result: integer},
		{action: game.SearchFrontAction{}, result: observation},
		{action: game.SearchNowAction{}, result: observation},
		{action: game.PrintIntegerAction{Integer: -42}, result: unit},
		{action: game.PrintBoolAction{Bool: true}, result: unit},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%T", tt.action), func(t *testing.T) {
			t.Parallel()
			boardCalls, robotCalls := 0, 0
			b := boardFunc{next: func(_ context.Context, reply game.Reply) (game.Decision, error) {
				boardCalls++
				switch boardCalls {
				case 1:
					if reply != nil {
						t.Error("initial reply must be nil")
					}
					return game.StartDecision{RobotID: "a"}, nil
				case 2:
					if reply != (game.ActionReply{RobotID: "a", Action: tt.action}) {
						t.Errorf("action or arguments lost in transit: %+v", reply)
					}
					return game.ResumeDecision{RobotID: "a", ReturnValue: tt.result}, nil
				case 3:
					if reply != (game.DoneReply{RobotID: "a"}) {
						t.Errorf("completion lost in transit: %+v", reply)
					}
					return game.DoneDecision{}, nil
				default:
					t.Error("unexpected board call")
					return nil, nil
				}
			}}
			r := robotFunc(func(_ context.Context, value game.Value) (game.Step, error) {
				robotCalls++
				if robotCalls == 1 {
					if value != nil {
						t.Error("initial return value must be nil")
					}
					return game.ActionStep{Action: tt.action}, nil
				}
				if value == nil || value != tt.result {
					t.Errorf("return value lost in transit: got %+v, want %+v", value, tt.result)
				}
				return game.DoneStep{}, nil
			})
			h := newHub(t, b, hub.Participant{ID: "a", Robot: r})
			report, err := h.Run(t.Context(), limits())
			if err != nil || report.Outcomes["a"] != (hub.CompletedOutcome{}) || report.Resumes != 2 || boardCalls != 3 || robotCalls != 2 {
				t.Errorf("unexpected result: %+v, %v", report, err)
			}
		})
	}
}

func TestTerrainObservationSurvivesBoardChanges(t *testing.T) {
	t.Parallel()
	var cell game.TerrainValue = game.LandTerrain{Kind: game.PlainLand{}, HasLuggage: true}
	observation := cell
	boardCalls, robotCalls := 0, 0
	b := boardFunc{next: func(_ context.Context, reply game.Reply) (game.Decision, error) {
		boardCalls++
		switch boardCalls {
		case 1:
			return game.StartDecision{RobotID: "a"}, nil
		case 2:
			if reply != (game.ActionReply{RobotID: "a", Action: game.SearchFrontAction{}}) {
				t.Error("expected search_front")
			}
			// Copy the observation before the live cell changes.
			result := cell
			cell = game.SeaTerrain{}
			return game.ResumeDecision{RobotID: "a", ReturnValue: result}, nil
		case 3:
			if reply != (game.DoneReply{RobotID: "a"}) {
				t.Errorf("expected completion after local terrain predicate, got %+v", reply)
			}
			return game.DoneDecision{}, nil
		default:
			t.Error("unexpected board call")
			return nil, errors.New("unexpected board call")
		}
	}}
	r := robotFunc(func(_ context.Context, value game.Value) (game.Step, error) {
		robotCalls++
		switch robotCalls {
		case 1:
			return game.ActionStep{Action: game.SearchFrontAction{}}, nil
		case 2:
			if value == nil {
				t.Error("expected terrain return value")
				return nil, errors.New("missing terrain return value")
			}
			terrain, ok := value.(game.TerrainValue)
			if !ok {
				t.Error("expected terrain return value")
				return nil, errors.New("unexpected terrain return value")
			}
			if terrain != observation {
				t.Errorf("terrain observation changed in transit: %+v", terrain)
			}
			// Terrain predicates are evaluated inside the robot without an action.
			if _, isLand := terrain.(game.LandTerrain); !isLand {
				t.Error("expected local is_land result to be true")
			}
			return game.DoneStep{}, nil
		default:
			t.Error("unexpected robot call")
			return nil, errors.New("unexpected robot call")
		}
	})
	h := newHub(t, b, hub.Participant{ID: "a", Robot: r})
	report, err := h.Run(t.Context(), limits())
	if err != nil {
		t.Error(err)
	}
	if report.Outcomes["a"] != (hub.CompletedOutcome{}) || report.Resumes != 2 || boardCalls != 3 || robotCalls != 2 {
		t.Errorf("local terrain predicate caused unexpected execution: %+v, board calls=%d, robot calls=%d", report, boardCalls, robotCalls)
	}
	if cell != (game.SeaTerrain{}) {
		t.Errorf("live cell did not change: %+v", cell)
	}
}

func TestStartsOnceThenResumesWithValue(t *testing.T) {
	t.Parallel()
	for _, value := range []game.Value{
		game.BoolValue{Bool: false}, game.IntegerValue{Integer: -42}, game.UnitValue{},
		game.LandTerrain{Kind: game.PlainLand{}}, game.LandTerrain{Kind: game.GoalLand{}}, game.LandTerrain{Kind: game.CheckpointLand{Order: 2}}, game.LandTerrain{Kind: game.PlainLand{}, HasLuggage: true}, game.RockTerrain{}, game.SeaTerrain{},
	} {
		t.Run(fmt.Sprintf("%T", value), func(t *testing.T) {
			t.Parallel()
			starts, resumes, boardCalls := 0, 0, 0
			r := robotFuncs{
				start: func(context.Context) (game.Step, error) {
					starts++
					return game.ActionStep{Action: game.SearchNowAction{}}, nil
				},
				next: func(_ context.Context, got game.Value) (game.Step, error) {
					resumes++
					if starts != 1 || got != value {
						t.Errorf("resumed after %d starts with %+v, want %+v", starts, got, value)
					}
					return game.DoneStep{}, nil
				},
			}
			b := boardFunc{next: func(_ context.Context, reply game.Reply) (game.Decision, error) {
				boardCalls++
				switch boardCalls {
				case 1:
					if reply != nil {
						t.Errorf("unexpected initial reply: %+v", reply)
					}
					return game.StartDecision{RobotID: "a"}, nil
				case 2:
					if reply != (game.ActionReply{RobotID: "a", Action: game.SearchNowAction{}}) {
						t.Errorf("unexpected action reply: %+v", reply)
					}
					return game.ResumeDecision{RobotID: "a", ReturnValue: value}, nil
				case 3:
					if reply != (game.DoneReply{RobotID: "a"}) {
						t.Errorf("unexpected completion reply: %+v", reply)
					}
					return game.DoneDecision{}, nil
				default:
					t.Error("unexpected board call")
					return nil, errors.New("no more board events")
				}
			}}
			report, err := newHub(t, b, hub.Participant{ID: "a", Robot: r}).Run(t.Context(), limits())
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
		events  []game.Decision
		started uint64
	}{
		{
			name:   "resume before start",
			events: []game.Decision{game.ResumeDecision{RobotID: "a", ReturnValue: game.UnitValue{}}},
		},
		{
			name:    "duplicate start",
			events:  []game.Decision{game.StartDecision{RobotID: "a"}, game.StartDecision{RobotID: "a"}},
			started: 1,
		},
		{
			name:    "missing return value",
			events:  []game.Decision{game.StartDecision{RobotID: "a"}, game.ResumeDecision{RobotID: "a"}},
			started: 1,
		},
		{
			name:    "typed nil return value",
			events:  []game.Decision{game.StartDecision{RobotID: "a"}, game.ResumeDecision{RobotID: "a", ReturnValue: (*game.BoolValue)(nil)}},
			started: 1,
		},
		{
			name:    "typed nil terrain",
			events:  []game.Decision{game.StartDecision{RobotID: "a"}, game.ResumeDecision{RobotID: "a", ReturnValue: (*game.LandTerrain)(nil)}},
			started: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var starts, resumes uint64
			index := 0
			r := robotFuncs{
				start: func(context.Context) (game.Step, error) {
					starts++
					return game.ActionStep{Action: game.DrillDurabilityAction{}}, nil
				},
				next: func(context.Context, game.Value) (game.Step, error) {
					resumes++
					return game.DoneStep{}, nil
				},
			}
			b := boardFunc{next: func(context.Context, game.Reply) (game.Decision, error) {
				if index >= len(tt.events) {
					t.Error("invalid request was not rejected")
					return nil, errors.New("no more board events")
				}
				decision := tt.events[index]
				index++
				return decision, nil
			}}
			report, err := newHub(t, b, hub.Participant{ID: "a", Robot: r}).Run(t.Context(), limits())
			if !errors.Is(err, hub.ErrProtocol) || starts != tt.started || resumes != 0 || report.Resumes != tt.started {
				t.Errorf("got %+v, %v with %d starts and %d resumes", report, err, starts, resumes)
			}
		})
	}
}
