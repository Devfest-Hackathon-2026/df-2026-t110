package hub_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game"
	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
)

type robotFunc func(context.Context, *game.Value) (game.Step, error)

func (f robotFunc) Next(ctx context.Context, value *game.Value) (game.Step, error) {
	return f(ctx, value)
}

type boardFunc struct {
	start func(context.Context, []game.RobotID) error
	next  func(context.Context, *game.Reply) (game.Decision, error)
}

func (b boardFunc) Start(ctx context.Context, ids []game.RobotID) error {
	if b.start != nil {
		return b.start(ctx, ids)
	}
	return nil
}

func (b boardFunc) Next(ctx context.Context, reply *game.Reply) (game.Decision, error) {
	return b.next(ctx, reply)
}

func limits() hub.Options { return hub.Options{MaxTicks: 10, MaxResumes: 100} }

func doneRobot() game.Robot {
	return robotFunc(func(context.Context, *game.Value) (game.Step, error) { return game.Step{Done: true}, nil })
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
		Terrain: [][]board.Terrain{{board.Land, board.Land, board.Land}},
		Robots: map[game.RobotID]board.RobotState{
			"a": {Position: board.Position{X: 0, Y: 0}, Direction: board.Right, DrillDurability: 3},
			"b": {Position: board.Position{X: 2, Y: 0}, Direction: board.Left, DrillDurability: 3},
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
	valueB := game.Value{Kind: game.Bool, Bool: true}
	valueA := game.Value{Kind: game.Unit}
	events := []game.Decision{
		{Request: &game.Request{RobotID: "b"}},
		{Request: &game.Request{RobotID: "a"}},
		{AdvanceTo: &two}, {Update: &game.TickUpdate{Tick: 2}},
		{Request: &game.Request{RobotID: "b", ReturnValue: &valueB}},
		{AdvanceTo: &five}, {Update: &game.TickUpdate{Tick: 5}},
		{Request: &game.Request{RobotID: "a", ReturnValue: &valueA}},
		{Done: true},
	}
	index := 0
	var calls []game.RobotID
	makeRobot := func(id game.RobotID, action game.Action, want game.Value) game.Robot {
		first := true
		return robotFunc(func(_ context.Context, value *game.Value) (game.Step, error) {
			calls = append(calls, id)
			if first {
				first = false
				if value != nil {
					t.Error("initial robot input must be nil")
				}
				return game.Step{Action: &action}, nil
			}
			if value == nil || *value != want {
				t.Errorf("robot %s got %+v, want %+v", id, value, want)
			}
			return game.Step{Done: true}, nil
		})
	}
	b := boardFunc{
		start: func(_ context.Context, ids []game.RobotID) error {
			if !reflect.DeepEqual(ids, []game.RobotID{"a", "b"}) {
				t.Errorf("registration order changed: %v", ids)
			}
			return nil
		},
		next: func(_ context.Context, reply *game.Reply) (game.Decision, error) {
			if index >= len(events) {
				t.Error("unexpected board call")
				return game.Decision{}, errors.New("no more board events")
			}
			if index > 0 && events[index-1].Request != nil {
				if reply == nil || reply.RobotID != events[index-1].Request.RobotID {
					t.Errorf("reply was not routed: %+v", reply)
				}
			} else if reply != nil {
				t.Errorf("unexpected reply: %+v", reply)
			}
			d := events[index]
			index++
			return d, nil
		},
	}
	h := newHub(t, b,
		hub.Participant{ID: "a", Robot: makeRobot("a", game.Action{Kind: game.Rotate, Rotation: game.Left}, valueA)},
		hub.Participant{ID: "b", Robot: makeRobot("b", game.Action{Kind: game.Forward}, valueB)},
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
	failed := robotFunc(func(context.Context, *game.Value) (game.Step, error) { return game.Step{}, failure })
	first := true
	other := robotFunc(func(_ context.Context, value *game.Value) (game.Step, error) {
		if first {
			first = false
			return game.Step{Action: &game.Action{Kind: game.Forward}}, nil
		}
		if value == nil || value.Kind != game.Bool || !value.Bool {
			t.Errorf("other robot got wrong result: %+v", value)
		}
		return game.Step{Done: true}, nil
	})
	h := newHub(t, b, hub.Participant{ID: "a", Robot: failed}, hub.Participant{ID: "b", Robot: other})
	report, err := h.Run(t.Context(), limits())
	if err != nil || report.Tick != 1 || !errors.Is(report.Outcomes["a"].Err, failure) || report.Outcomes["a"].Status != hub.Failed || report.Outcomes["b"].Status != hub.Completed {
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
		query := robotFunc(func(context.Context, *game.Value) (game.Step, error) {
			return game.Step{Action: &game.Action{Kind: game.DrillDurability}}, nil
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
		move := robotFunc(func(context.Context, *game.Value) (game.Step, error) {
			return game.Step{Action: &game.Action{Kind: game.Rotate, Rotation: game.Right}}, nil
		})
		h := newHub(t, b, hub.Participant{ID: "a", Robot: move}, hub.Participant{ID: "b", Robot: move})
		report, err := h.Run(t.Context(), hub.Options{MaxTicks: 1, MaxResumes: 100})
		if !errors.Is(err, hub.ErrTickLimit) || report.Tick != 1 || b.Snapshot().Tick != 1 || b.Snapshot().Robots["a"].Direction != board.Down {
			t.Errorf("got %+v, %v, board %+v", report, err, b.Snapshot())
		}
	})
	t.Run("future tick leap rejected", func(t *testing.T) {
		t.Parallel()
		calls := 0
		five := uint64(5)
		b := boardFunc{next: func(context.Context, *game.Reply) (game.Decision, error) {
			calls++
			return game.Decision{AdvanceTo: &five}, nil
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
		"empty":              {{}},
		"multiple events":    {{Done: true, Request: &game.Request{RobotID: "a"}}},
		"unknown robot":      {{Request: &game.Request{RobotID: "unknown"}}},
		"finished robot":     {{Request: &game.Request{RobotID: "a"}}, {Request: &game.Request{RobotID: "a"}}},
		"early completion":   {{Done: true}},
		"unannounced update": {{Update: &game.TickUpdate{Tick: 1}}},
		"wrong commit tick":  {{AdvanceTo: &one}, {Update: &game.TickUpdate{Tick: 2}}},
		"missing commit":     {{AdvanceTo: &one}, {Request: &game.Request{RobotID: "a"}}},
		"nonincreasing tick": {{AdvanceTo: &two}, {Update: &game.TickUpdate{Tick: 2}}, {AdvanceTo: &one}},
	}
	for name, events := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			i := 0
			b := boardFunc{next: func(context.Context, *game.Reply) (game.Decision, error) {
				if i >= len(events) {
					t.Error("protocol violation was not rejected")
					return game.Decision{}, errors.New("no more board events")
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
			next: func(context.Context, *game.Reply) (game.Decision, error) { return game.Decision{}, failure },
		}
		h := newHub(t, b, hub.Participant{ID: "a", Robot: doneRobot()})
		if _, err := h.Run(t.Context(), limits()); !errors.Is(err, failure) {
			t.Errorf("board error lost: %v", err)
		}
	}
	b := grid(t)
	move := robotFunc(func(context.Context, *game.Value) (game.Step, error) {
		return game.Step{Action: &game.Action{Kind: game.Forward}}, nil
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
	move := robotFunc(func(context.Context, *game.Value) (game.Step, error) {
		return game.Step{Action: &game.Action{Kind: game.Forward}}, nil
	})
	interrupted := robotFunc(func(context.Context, *game.Value) (game.Step, error) {
		cancel()
		return game.Step{}, context.Canceled
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
	for _, step := range []game.Step{{}, {Done: true, Action: &game.Action{Kind: game.Forward}}} {
		r := robotFunc(func(context.Context, *game.Value) (game.Step, error) { return step, nil })
		h := newHub(t, grid(t), hub.Participant{ID: "a", Robot: r}, hub.Participant{ID: "b", Robot: doneRobot()})
		report, err := h.Run(t.Context(), limits())
		if err != nil || report.Outcomes["a"].Status != hub.Failed || !errors.Is(report.Outcomes["a"].Err, hub.ErrProtocol) {
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
	field := game.FieldRef("board-a:observation-1")
	boolean := game.Value{Kind: game.Bool, Bool: true}
	integer := game.Value{Kind: game.Integer, Integer: 3}
	unit := game.Value{Kind: game.Unit}
	observation := game.Value{Kind: game.Field, Field: field}
	tests := []struct {
		action game.Action
		result game.Value
	}{
		{game.Action{Kind: game.Forward}, boolean},
		{game.Action{Kind: game.Back}, boolean},
		{game.Action{Kind: game.Rotate, Rotation: game.Left}, unit},
		{game.Action{Kind: game.Wait, Integer: 7}, unit},
		{game.Action{Kind: game.PickUp}, boolean},
		{game.Action{Kind: game.Release}, boolean},
		{game.Action{Kind: game.Drill}, boolean},
		{game.Action{Kind: game.DrillDurability}, integer},
		{game.Action{Kind: game.Distance}, integer},
		{game.Action{Kind: game.SearchFront}, observation},
		{game.Action{Kind: game.SearchNow}, observation},
		{game.Action{Kind: game.IsLand, Field: field}, boolean},
		{game.Action{Kind: game.IsGoal, Field: field}, boolean},
		{game.Action{Kind: game.IsCheckpoint, Field: field}, boolean},
		{game.Action{Kind: game.IsLaggage, Field: field}, boolean},
		{game.Action{Kind: game.IsSea, Field: field}, boolean},
		{game.Action{Kind: game.IsRock, Field: field}, boolean},
		{game.Action{Kind: game.PrintInteger, Integer: -42}, unit},
		{game.Action{Kind: game.PrintBool, Bool: true}, unit},
	}
	for _, tt := range tests {
		t.Run(string(tt.action.Kind), func(t *testing.T) {
			t.Parallel()
			boardCalls, robotCalls := 0, 0
			b := boardFunc{next: func(_ context.Context, reply *game.Reply) (game.Decision, error) {
				boardCalls++
				switch boardCalls {
				case 1:
					if reply != nil {
						t.Error("initial reply must be nil")
					}
					return game.Decision{Request: &game.Request{RobotID: "a"}}, nil
				case 2:
					if reply == nil || reply.RobotID != "a" || reply.Action == nil || *reply.Action != tt.action || reply.Done || reply.Err != nil {
						t.Errorf("action or arguments lost in transit: %+v", reply)
					}
					return game.Decision{Request: &game.Request{RobotID: "a", ReturnValue: &tt.result}}, nil
				case 3:
					if reply == nil || reply.RobotID != "a" || !reply.Done || reply.Action != nil || reply.Err != nil {
						t.Errorf("completion lost in transit: %+v", reply)
					}
					return game.Decision{Done: true}, nil
				default:
					t.Error("unexpected board call")
					return game.Decision{}, nil
				}
			}}
			r := robotFunc(func(_ context.Context, value *game.Value) (game.Step, error) {
				robotCalls++
				if robotCalls == 1 {
					if value != nil {
						t.Error("initial return value must be nil")
					}
					return game.Step{Action: &tt.action}, nil
				}
				if value == nil || *value != tt.result {
					t.Errorf("return value lost in transit: got %+v, want %+v", value, tt.result)
				}
				return game.Step{Done: true}, nil
			})
			h := newHub(t, b, hub.Participant{ID: "a", Robot: r})
			report, err := h.Run(t.Context(), limits())
			if err != nil || report.Outcomes["a"].Status != hub.Completed || report.Resumes != 2 || boardCalls != 3 || robotCalls != 2 {
				t.Errorf("unexpected result: %+v, %v", report, err)
			}
		})
	}
}

func TestFieldObservationRoundTrip(t *testing.T) {
	t.Parallel()
	observation := game.FieldRef("board-a:immutable-observation")
	boardCalls, robotCalls := 0, 0
	b := boardFunc{next: func(_ context.Context, reply *game.Reply) (game.Decision, error) {
		boardCalls++
		switch boardCalls {
		case 1:
			return game.Decision{Request: &game.Request{RobotID: "a"}}, nil
		case 2:
			if reply == nil || reply.Action == nil || reply.Action.Kind != game.SearchFront {
				t.Error("expected search_front")
			}
			return game.Decision{Request: &game.Request{RobotID: "a", ReturnValue: &game.Value{Kind: game.Field, Field: observation}}}, nil
		case 3:
			if reply == nil || reply.Action == nil || reply.Action.Kind != game.IsLand || reply.Action.Field != observation {
				t.Errorf("field reference not returned to issuing board: %+v", reply)
			}
			return game.Decision{Request: &game.Request{RobotID: "a", ReturnValue: &game.Value{Kind: game.Bool, Bool: true}}}, nil
		default:
			return game.Decision{Done: true}, nil
		}
	}}
	r := robotFunc(func(_ context.Context, value *game.Value) (game.Step, error) {
		robotCalls++
		switch robotCalls {
		case 1:
			return game.Step{Action: &game.Action{Kind: game.SearchFront}}, nil
		case 2:
			if value == nil {
				t.Error("expected field return value")
				return game.Step{}, errors.New("missing field return value")
			}
			if value.Kind != game.Field {
				t.Error("expected field return value")
			}
			return game.Step{Action: &game.Action{Kind: game.IsLand, Field: value.Field}}, nil
		default:
			if value == nil || value.Kind != game.Bool || !value.Bool {
				t.Error("expected predicate result")
			}
			return game.Step{Done: true}, nil
		}
	})
	h := newHub(t, b, hub.Participant{ID: "a", Robot: r})
	if _, err := h.Run(t.Context(), limits()); err != nil {
		t.Error(err)
	}
}
