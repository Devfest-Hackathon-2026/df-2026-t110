package hub_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	"github.com/dvpico/Programming-Learning-AR/Server/game/recorder"
)

// recordingSpy observes deliveries without implementing history storage.
type recordingSpy struct{ messages []gameevent.GameEvent }

func (r *recordingSpy) Record(event gameevent.GameEvent) error {
	r.messages = append(r.messages, event)
	return nil
}

func TestIndependentEventMessages(t *testing.T) {
	t.Parallel()
	current := robotevent.CurrentNodeEvent{HintID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}
	init := robotevent.VariableInitEvent{HintID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", Value: values.StackRecordValue{Elements: []values.RecordValue{values.IntegerValue{Integer: 1}}}}
	update := robotevent.VariableUpdateEvent{HintID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", Diff: []robotevent.ValueUpdate{robotevent.StackUpdate{Update: robotevent.PushBack{Value: values.IntegerValue{Integer: 2}}}}}
	drop := robotevent.VariableDropEvent{HintID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"}
	a := robotFuncs{
		start: func() (robotmsg.Step, error) {
			return robotmsg.ActionStep{Action: actions.ForwardAction{}}, nil
		},
		next: func(value values.Value) (robotmsg.Step, error) {
			if value != (values.BoolValue{Bool: true}) {
				t.Errorf("wrong gameplay return value: %+v", value)
			}
			return robotmsg.DoneStep{}, nil
		},
		startEvents: []robotevent.PicolangEvent{current, init},
		nextEvents:  []robotevent.PicolangEvent{update, drop},
	}
	b := robotFuncs{
		start:       func() (robotmsg.Step, error) { return robotmsg.DoneStep{}, nil },
		startEvents: []robotevent.PicolangEvent{current},
	}
	var received recorder.Recorder
	h := newHub(t, grid(t), hub.Participant{ID: "a", Robot: a}, hub.Participant{ID: "b", Robot: b})
	options := limits()
	options.Recorder = &received
	report, err := h.Run(options)
	if err != nil || report.Resumes() != 3 {
		t.Errorf("run = %+v, %v", report, err)
	}
	plain := fieldevent.Cell{Type: "land", Kind: &fieldevent.LandKind{Type: "plain"}}
	want := []gameevent.GameEvent{
		gameevent.FieldEvent{Event: fieldevent.FieldEvent{State: fieldevent.InitialState{
			Terrain: [][]fieldevent.Cell{{plain, plain, plain}},
			Robots: []fieldevent.RobotState{
				{ID: "a", Direction: "east", DrillDurability: 3},
				{ID: "b", Position: fieldevent.Position{X: 2}, Direction: "west", DrillDurability: 3},
			},
		}}},
		gameevent.RobotEvent{InterpreterID: "a", Event: current},
		gameevent.RobotEvent{InterpreterID: "a", Event: init},
		gameevent.RobotEvent{InterpreterID: "b", Event: current},
		gameevent.FieldEvent{Event: fieldevent.FieldEvent{Tick: 1, State: fieldevent.UpdateState{
			Actions: []fieldevent.ActionResult{
				fieldevent.MoveResult{Kind: fieldevent.Forward, RobotID: "a", Success: true, To: &fieldevent.Position{X: 1}},
			},
			RemovedRobotIDs: []protocol.RobotID{"b"},
		}}},
		gameevent.RobotEvent{InterpreterID: "a", Event: update},
		gameevent.RobotEvent{InterpreterID: "a", Event: drop},
		gameevent.FieldEvent{Event: fieldevent.FieldEvent{Tick: 1, State: fieldevent.UpdateState{
			RemovedRobotIDs: []protocol.RobotID{"a"},
		}}},
	}
	if !reflect.DeepEqual(received.Events, want) {
		t.Errorf("deliveries = %+v, want %+v", received.Events, want)
	}
}

type failingRecorder struct {
	calls int
	err   error
}

func (r *failingRecorder) Record(gameevent.GameEvent) error {
	r.calls++
	return r.err
}

func TestRecordingFailureStopsSuccessfulRobotReply(t *testing.T) {
	t.Parallel()
	boardCalls := 0
	b := boardFunc{next: func(robotmsg.Reply) (boardmsg.Decision, error) {
		boardCalls++
		if boardCalls != 1 {
			t.Error("robot reply delivered after recording failure")
		}
		return boardmsg.StartDecision{RobotID: "a"}, nil
	}}
	r := robotFuncs{
		start:       func() (robotmsg.Step, error) { return robotmsg.DoneStep{}, nil },
		startEvents: []robotevent.PicolangEvent{robotevent.CurrentNodeEvent{HintID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}},
	}
	storageErr := errors.New("storage failed")
	recorder := failingRecorder{err: storageErr}
	options := limits()
	options.Recorder = &recorder
	report, err := newHub(t, b, hub.Participant{ID: "a", Robot: r}).Run(options)
	if !errors.Is(err, hub.ErrRecording) || !errors.Is(err, storageErr) || boardCalls != 1 || recorder.calls != 1 || report.Outcome("a") != (hub.StoppedOutcome{}) {
		t.Errorf("run continued after recording failure: %+v, %v", report, err)
	}
}

func TestMessagesFromFailedCallsAndRecordingFailures(t *testing.T) {
	t.Parallel()
	for _, source := range []string{"board start", "board next", "robot start", "robot next"} {
		for _, recordingFails := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/recording fails=%t", source, recordingFails), func(t *testing.T) {
				t.Parallel()
				producerErr := errors.New("producer failed")
				recordingErr := errors.New("storage failed")
				field := []fieldevent.FieldEvent{{State: fieldevent.UpdateState{}}}
				debug := []robotevent.PicolangEvent{robotevent.CurrentNodeEvent{HintID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}}
				boardCalls, robotCalls := 0, 0
				b := boardFunc{
					next: func(reply robotmsg.Reply) (boardmsg.Decision, error) {
						boardCalls++
						switch {
						case source == "board next":
							return nil, producerErr
						case boardCalls == 1:
							return boardmsg.StartDecision{RobotID: "a"}, nil
						case source == "robot next" && boardCalls == 2:
							return boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.UnitValue{}}, nil
						default:
							failure, ok := reply.(robotmsg.ErrorReply)
							if !ok || !errors.Is(failure.Err, producerErr) {
								t.Errorf("producer error was not delivered to board: %+v", reply)
							}
							return boardmsg.NewDoneDecision(nil, false), nil
						}
					},
				}
				r := robotFuncs{
					start: func() (robotmsg.Step, error) {
						robotCalls++
						if source == "robot start" {
							return nil, producerErr
						}
						return robotmsg.ActionStep{Action: actions.RotateAction{Rotation: values.Left}}, nil
					},
					next: func(values.Value) (robotmsg.Step, error) {
						robotCalls++
						return nil, producerErr
					},
				}
				switch source {
				case "board start":
					b.start = func([]protocol.RobotID) error { return producerErr }
					b.startEvents = field
				case "board next":
					b.nextEvents = field
				case "robot start":
					r.startEvents = debug
				case "robot next":
					r.nextEvents = debug
				}
				var received recordingSpy
				failed := failingRecorder{err: recordingErr}
				options := limits()
				options.Recorder = &received
				if recordingFails {
					options.Recorder = &failed
				}
				h := newHub(t, b, hub.Participant{ID: "a", Robot: r})
				report, err := h.Run(options)
				if recordingFails {
					if !errors.Is(err, hub.ErrRecording) || !errors.Is(err, recordingErr) || !errors.Is(err, producerErr) || failed.calls != 1 {
						t.Errorf("recording failure = %v, calls %d", err, failed.calls)
					}
					wantBoardCalls := map[string]int{"board start": 0, "board next": 1, "robot start": 1, "robot next": 2}[source]
					if boardCalls != wantBoardCalls {
						t.Errorf("routing continued after recording failure: %d board calls", boardCalls)
					}
				} else {
					if source == "board start" || source == "board next" {
						if !errors.Is(err, producerErr) || robotCalls != 0 {
							t.Errorf("board error = %v, robot calls %d", err, robotCalls)
						}
					} else if failure, ok := report.Outcome("a").(hub.FailedOutcome); err != nil || !ok || !errors.Is(failure.Err, producerErr) {
						t.Errorf("robot failure = %+v, %v", report, err)
					}
					if len(received.messages) != 1 {
						t.Errorf("failed call's message lost: %+v", received.messages)
					}
				}
			})
		}
	}
}

func TestReturnedRecordsPrecedeProtocolChecks(t *testing.T) {
	t.Parallel()
	for _, source := range []string{"board", "robot"} {
		t.Run(source, func(t *testing.T) {
			t.Parallel()
			var received recordingSpy
			robotCalls := 0
			b := boardFunc{next: func(reply robotmsg.Reply) (boardmsg.Decision, error) {
				if source == "board" {
					return nil, nil
				}
				if robotCalls == 0 {
					return boardmsg.StartDecision{RobotID: "a"}, nil
				}
				return boardmsg.NewDoneDecision(nil, false), nil
			}}
			r := robotFuncs{start: func() (robotmsg.Step, error) {
				robotCalls++
				return nil, nil
			}}
			if source == "board" {
				b.nextEvents = []fieldevent.FieldEvent{{State: fieldevent.UpdateState{}}}
			} else {
				r.startEvents = []robotevent.PicolangEvent{robotevent.CurrentNodeEvent{HintID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}}
			}
			options := limits()
			options.Recorder = &received
			report, err := newHub(t, b, hub.Participant{ID: "a", Robot: r}).Run(options)
			if source == "board" {
				if !errors.Is(err, hub.ErrProtocol) {
					t.Errorf("invalid decision accepted: %v", err)
				}
			} else if failure, ok := report.Outcome("a").(hub.FailedOutcome); err != nil || !ok || !errors.Is(failure.Err, hub.ErrProtocol) {
				t.Errorf("invalid step accepted: %+v, %v", report, err)
			}
			if len(received.messages) != 1 {
				t.Errorf("returned record lost: %+v", received.messages)
			}
		})
	}
}
