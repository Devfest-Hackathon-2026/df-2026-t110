// Package hub routes board decisions and robot replies without interpreting actions.
package hub

import (
	"errors"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/game"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

var (
	ErrRecording   = errors.New("recording failed")
	ErrResumeLimit = errors.New("robot resume limit reached")
	ErrProtocol    = errors.New("invalid game protocol")
)

type Participant struct {
	ID    protocol.RobotID
	Robot game.Robot
}

type Options struct {
	// MaxResumes counts both Start and Next calls.
	MaxResumes uint64
	Recorder   game.Recorder
}

type Report struct {
	Resumes  uint64
	Outcomes map[protocol.RobotID]Outcome
}

type Hub struct {
	board        game.Board
	participants []Participant
	used         bool
}

func New(board game.Board, participants []Participant) (*Hub, error) {
	if board == nil || len(participants) == 0 {
		return nil, errors.New("board and at least one robot are required")
	}
	seen := make(map[protocol.RobotID]struct{}, len(participants))
	for _, p := range participants {
		if _, exists := seen[p.ID]; p.ID == "" || p.Robot == nil || exists {
			return nil, fmt.Errorf("invalid or duplicate robot %q", p.ID)
		}
		seen[p.ID] = struct{}{}
	}
	return &Hub{board: board, participants: append([]Participant(nil), participants...)}, nil
}

func (h *Hub) Run(options Options) (Report, error) {
	report := Report{Outcomes: make(map[protocol.RobotID]Outcome, len(h.participants))}
	if h.used || options.MaxResumes == 0 {
		return report, errors.New("hub is single-use and limits must be positive")
	}
	h.used = true
	robots := make(map[protocol.RobotID]game.Robot, len(h.participants))
	ids := make([]protocol.RobotID, 0, len(h.participants))
	for _, p := range h.participants {
		robots[p.ID] = p.Robot
		ids = append(ids, p.ID)
		report.Outcomes[p.ID] = UnstartedOutcome{}
	}
	events, err := h.board.Start(ids)
	if options.Recorder != nil {
		for _, event := range events {
			if recordErr := options.Recorder.Record(gameevent.FieldEvent{Event: event}); recordErr != nil {
				return report, errors.Join(err, fmt.Errorf("%w: %w", ErrRecording, recordErr))
			}
		}
	}
	if err != nil {
		return report, fmt.Errorf("start board: %w", err)
	}
	var reply robotmsg.Reply
	for {
		output, err := h.board.Next(reply)
		if options.Recorder != nil {
			for _, event := range output.Events {
				if recordErr := options.Recorder.Record(gameevent.FieldEvent{Event: event}); recordErr != nil {
					return report, errors.Join(err, fmt.Errorf("%w: %w", ErrRecording, recordErr))
				}
			}
		}
		if err != nil {
			return report, fmt.Errorf("board.Next: %w", err)
		}
		reply = nil
		switch decision := output.Decision.(type) {
		case boardmsg.StartDecision, boardmsg.ResumeDecision:
			var id protocol.RobotID
			switch request := decision.(type) {
			case boardmsg.StartDecision:
				id = request.RobotID
			case boardmsg.ResumeDecision:
				id = request.RobotID
			}
			robot, exists := robots[id]
			if !exists {
				return report, fmt.Errorf("%w: unavailable robot %q", ErrProtocol, id)
			}
			switch request := decision.(type) {
			case boardmsg.StartDecision:
				if _, fresh := report.Outcomes[id].(UnstartedOutcome); !fresh {
					return report, fmt.Errorf("%w: robot %q already started", ErrProtocol, id)
				}
			case boardmsg.ResumeDecision:
				if _, stopped := report.Outcomes[id].(StoppedOutcome); !stopped || !values.ValidValue(request.ReturnValue) {
					return report, fmt.Errorf("%w: invalid resume for robot %q", ErrProtocol, id)
				}
			}
			if report.Resumes == options.MaxResumes {
				return report, ErrResumeLimit
			}
			report.Resumes++
			report.Outcomes[id] = StoppedOutcome{}
			var output robotmsg.RobotOutput
			var err error
			switch request := decision.(type) {
			case boardmsg.StartDecision:
				output, err = robot.Start()
			case boardmsg.ResumeDecision:
				output, err = robot.Next(request.ReturnValue)
			}
			if options.Recorder != nil {
				for _, event := range output.Events {
					if recordErr := options.Recorder.Record(gameevent.RobotEvent{InterpreterID: id, Event: event}); recordErr != nil {
						return report, errors.Join(err, fmt.Errorf("%w: %w", ErrRecording, recordErr))
					}
				}
			}
			if err == nil {
				switch step := output.Step.(type) {
				case robotmsg.ActionStep:
					if !actions.ValidAction(step.Action) {
						err = fmt.Errorf("%w: invalid robot action %T", ErrProtocol, step.Action)
					} else {
						reply = robotmsg.ActionReply{RobotID: id, Action: step.Action}
					}
				case robotmsg.DoneStep:
					reply = robotmsg.DoneReply{RobotID: id}
					report.Outcomes[id] = CompletedOutcome{}
				default:
					err = fmt.Errorf("%w: invalid robot step %T", ErrProtocol, output.Step)
				}
			}
			if err != nil {
				reply = robotmsg.ErrorReply{RobotID: id, Err: err}
				report.Outcomes[id] = FailedOutcome{Err: err}
			}
		case boardmsg.DoneDecision:
			for _, outcome := range report.Outcomes {
				switch outcome.(type) {
				case UnstartedOutcome, StoppedOutcome:
					return report, fmt.Errorf("%w: board finished with active robots", ErrProtocol)
				}
			}
			return report, nil
		default:
			return report, fmt.Errorf("%w: invalid board decision %T", ErrProtocol, decision)
		}
	}
}
