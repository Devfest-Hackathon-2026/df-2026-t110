package board_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func TestWaitDurationAndTickLimit(t *testing.T) {
	t.Parallel()

	resume := boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.UnitValue{}}
	tests := []struct {
		name     string
		ticks    int64
		limit    uint64
		wantTick uint64
		wantErr  error
		want     boardmsg.BoardOutput
	}{
		{
			name:  "negative waits return immediately",
			ticks: math.MinInt64,
			want:  boardmsg.BoardOutput{Decision: resume},
		},
		{
			name: "zero waits return immediately",
			want: boardmsg.BoardOutput{Decision: resume},
		},
		{
			name:     "one turn at the tick limit",
			ticks:    1,
			limit:    1,
			wantTick: 1,
			want: boardmsg.BoardOutput{
				Decision: resume,
				Events: []fieldevent.FieldEvent{{Tick: 1, State: fieldevent.UpdateState{
					Actions: []fieldevent.ActionResult{fieldevent.WaitResult{RobotID: "a", Ticks: 1}},
				}}},
			},
		},
		{
			name:     "large idle span is compact",
			ticks:    math.MaxInt64,
			wantTick: math.MaxInt64,
			want: boardmsg.BoardOutput{
				Decision: resume,
				Events: []fieldevent.FieldEvent{
					{Tick: 1, State: fieldevent.UpdateState{Actions: []fieldevent.ActionResult{fieldevent.WaitResult{RobotID: "a", Ticks: math.MaxInt64}}}},
					{Tick: math.MaxInt64, State: fieldevent.UpdateState{}},
				},
			},
		},
		{
			name:     "idle advance stops at the tick limit",
			ticks:    math.MaxInt64,
			limit:    3,
			wantTick: 3,
			wantErr:  board.ErrTickLimit,
			want: boardmsg.BoardOutput{Events: []fieldevent.FieldEvent{
				{Tick: 1, State: fieldevent.UpdateState{Actions: []fieldevent.ActionResult{fieldevent.WaitResult{RobotID: "a", Ticks: math.MaxInt64}}}},
				{Tick: 3, State: fieldevent.UpdateState{}},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reply := robotmsg.ActionReply{RobotID: "a", Action: actions.WaitAction{Integer: tt.ticks}}
			config, err := board.NewConfig(
				land(1, 1),
				map[protocol.RobotID]board.RobotState{"a": state(0, 0, board.East)},
				tt.limit,
				board.AllStoppedRules{},
			)
			if err != nil {
				t.Fatal(err)
			}
			b := newBoard(t, config, []protocol.RobotID{"a"})
			wantState := b.Snapshot()
			wantState.Tick = tt.wantTick
			next(t, b, nil)

			output, err := b.Next(reply)
			snapshot := b.Snapshot()

			if !errors.Is(err, tt.wantErr) || !reflect.DeepEqual(output, tt.want) {
				t.Errorf("wait = %+v, %v, want %+v, %v", output, err, tt.want, tt.wantErr)
			}
			if !reflect.DeepEqual(snapshot, wantState) {
				t.Errorf("state after wait = %+v, want %+v", snapshot, wantState)
			}
		})
	}
}

func TestWaitKeepsCellOccupiedAndLetsOtherRobotsAct(t *testing.T) {
	t.Parallel()

	wait := robotmsg.ActionReply{RobotID: "a", Action: actions.WaitAction{Integer: 3}}
	forward := robotmsg.ActionReply{RobotID: "b", Action: actions.ForwardAction{}}
	wantBlocked := boardmsg.ResumeDecision{RobotID: "b", ReturnValue: values.BoolValue{Bool: false}}
	wantWoken := boardmsg.ResumeDecision{RobotID: "a", ReturnValue: values.UnitValue{}}
	wantIdle := []fieldevent.FieldEvent{
		{Tick: 2, State: fieldevent.UpdateState{RemovedRobotIDs: []protocol.RobotID{"b"}}},
		{Tick: 3, State: fieldevent.UpdateState{}},
	}
	config, err := board.NewConfig(
		land(2, 1),
		map[protocol.RobotID]board.RobotState{
			"a": state(1, 0, board.East),
			"b": state(0, 0, board.East),
		},
		0,
		board.AllStoppedRules{},
	)
	if err != nil {
		t.Fatal(err)
	}
	b := newBoard(t, config, []protocol.RobotID{"a", "b"})
	next(t, b, nil)

	next(t, b, wait)
	blocked := next(t, b, forward)
	afterMove := b.Snapshot()
	woken, err := b.Next(robotmsg.DoneReply{RobotID: "b"})
	afterWait := b.Snapshot()

	if blocked != wantBlocked || afterMove.Tick != 1 || afterMove.Robots["b"].Position() != board.NewPosition(0, 0) {
		t.Errorf("move into a waiting robot's cell = %+v, board = %+v", blocked, afterMove)
	}
	if err != nil || woken.Decision != wantWoken || !reflect.DeepEqual(woken.Events, wantIdle) || afterWait.Tick != 3 || len(afterWait.Robots) != 1 {
		t.Errorf("wait after other robot stops = %+v, %v, board = %+v", woken, err, afterWait)
	}
}
