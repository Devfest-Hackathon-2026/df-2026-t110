// Package hub routes board decisions and robot replies without interpreting actions.
package hub

import (
	"context"
	"errors"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/game"
)

var (
	ErrTickLimit   = errors.New("tick limit reached")
	ErrResumeLimit = errors.New("robot resume limit reached")
	ErrProtocol    = errors.New("invalid game protocol")
)

type Participant struct {
	ID    game.RobotID
	Robot game.Robot
}

type Options struct {
	MaxTicks uint64
	// MaxResumes counts both Start and Next calls.
	MaxResumes uint64
	OnTick     func(game.TickUpdate) error
}

type Report struct {
	Tick     uint64
	Resumes  uint64
	Outcomes map[game.RobotID]Outcome
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
	seen := make(map[game.RobotID]struct{}, len(participants))
	for _, p := range participants {
		if _, exists := seen[p.ID]; p.ID == "" || p.Robot == nil || exists {
			return nil, fmt.Errorf("invalid or duplicate robot %q", p.ID)
		}
		seen[p.ID] = struct{}{}
	}
	return &Hub{board: board, participants: append([]Participant(nil), participants...)}, nil
}

func (h *Hub) Run(ctx context.Context, options Options) (Report, error) {
	report := Report{Outcomes: make(map[game.RobotID]Outcome, len(h.participants))}
	if h.used || options.MaxTicks == 0 || options.MaxResumes == 0 {
		return report, errors.New("hub is single-use and limits must be positive")
	}
	h.used = true
	robots := make(map[game.RobotID]game.Robot, len(h.participants))
	ids := make([]game.RobotID, 0, len(h.participants))
	for _, p := range h.participants {
		robots[p.ID] = p.Robot
		ids = append(ids, p.ID)
		report.Outcomes[p.ID] = UnstartedOutcome{}
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	if err := h.board.Start(ctx, ids); err != nil {
		return report, fmt.Errorf("start board: %w", err)
	}
	var reply game.Reply
	var proposed *uint64
	for {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		decision, err := h.board.Next(ctx, reply)
		if err != nil {
			return report, fmt.Errorf("advance board: %w", err)
		}
		reply = nil
		if proposed != nil {
			if _, committing := decision.(game.UpdateDecision); !committing {
				return report, fmt.Errorf("%w: expected tick commit", ErrProtocol)
			}
		}
		switch decision := decision.(type) {
		case game.AdvanceDecision:
			tick := decision.Tick
			if tick <= report.Tick {
				return report, fmt.Errorf("%w: tick must increase", ErrProtocol)
			}
			if tick > options.MaxTicks {
				return report, ErrTickLimit
			}
			proposed = &tick
		case game.UpdateDecision:
			if proposed == nil || decision.Update.Tick != *proposed {
				return report, fmt.Errorf("%w: unexpected tick commit", ErrProtocol)
			}
			proposed = nil
			report.Tick = decision.Update.Tick
			if options.OnTick != nil {
				if err := options.OnTick(decision.Update); err != nil {
					return report, fmt.Errorf("tick callback: %w", err)
				}
			}
		case game.StartDecision, game.ResumeDecision:
			var id game.RobotID
			switch request := decision.(type) {
			case game.StartDecision:
				id = request.RobotID
			case game.ResumeDecision:
				id = request.RobotID
			}
			robot, exists := robots[id]
			if !exists {
				return report, fmt.Errorf("%w: unavailable robot %q", ErrProtocol, id)
			}
			switch request := decision.(type) {
			case game.StartDecision:
				if _, fresh := report.Outcomes[id].(UnstartedOutcome); !fresh {
					return report, fmt.Errorf("%w: robot %q already started", ErrProtocol, id)
				}
			case game.ResumeDecision:
				if _, stopped := report.Outcomes[id].(StoppedOutcome); !stopped || !game.ValidValue(request.ReturnValue) {
					return report, fmt.Errorf("%w: invalid resume for robot %q", ErrProtocol, id)
				}
			}
			if report.Resumes == options.MaxResumes {
				return report, ErrResumeLimit
			}
			report.Resumes++
			report.Outcomes[id] = StoppedOutcome{}
			var step game.Step
			var err error
			switch request := decision.(type) {
			case game.StartDecision:
				step, err = robot.Start(ctx)
			case game.ResumeDecision:
				step, err = robot.Next(ctx, request.ReturnValue)
			}
			if cancelErr := ctx.Err(); cancelErr != nil {
				return report, cancelErr
			}
			if err == nil {
				switch step := step.(type) {
				case game.ActionStep:
					if !game.ValidAction(step.Action) {
						err = fmt.Errorf("%w: invalid robot action %T", ErrProtocol, step.Action)
					} else {
						reply = game.ActionReply{RobotID: id, Action: step.Action}
					}
				case game.DoneStep:
					reply = game.DoneReply{RobotID: id}
					report.Outcomes[id] = CompletedOutcome{}
				default:
					err = fmt.Errorf("%w: invalid robot step %T", ErrProtocol, step)
				}
			}
			if err != nil {
				reply = game.ErrorReply{RobotID: id, Err: err}
				report.Outcomes[id] = FailedOutcome{Err: err}
			}
		case game.DoneDecision:
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
