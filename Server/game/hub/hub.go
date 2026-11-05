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
	MaxTicks   uint64
	MaxResumes uint64
	OnTick     func(game.TickUpdate) error
}

type Status string

const (
	Stopped   Status = "stopped"
	Completed Status = "completed"
	Failed    Status = "failed"
)

type Outcome struct {
	Status Status
	Err    error
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
	seen := make(map[game.RobotID]bool, len(participants))
	for _, p := range participants {
		if p.ID == "" || p.Robot == nil || seen[p.ID] {
			return nil, fmt.Errorf("invalid or duplicate robot %q", p.ID)
		}
		seen[p.ID] = true
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
	resumed := make(map[game.RobotID]bool, len(h.participants))
	ids := make([]game.RobotID, 0, len(h.participants))
	for _, p := range h.participants {
		robots[p.ID] = p.Robot
		ids = append(ids, p.ID)
		report.Outcomes[p.ID] = Outcome{Status: Stopped}
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	if err := h.board.Start(ctx, ids); err != nil {
		return report, fmt.Errorf("start board: %w", err)
	}
	var reply *game.Reply
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
		if !validDecision(decision) {
			return report, ErrProtocol
		}
		if proposed != nil && decision.Update == nil {
			return report, fmt.Errorf("%w: expected tick commit", ErrProtocol)
		}
		switch {
		case decision.AdvanceTo != nil:
			tick := *decision.AdvanceTo
			if tick <= report.Tick {
				return report, fmt.Errorf("%w: tick must increase", ErrProtocol)
			}
			if tick > options.MaxTicks {
				return report, ErrTickLimit
			}
			proposed = &tick
		case decision.Update != nil:
			if proposed == nil || decision.Update.Tick != *proposed {
				return report, fmt.Errorf("%w: unexpected tick commit", ErrProtocol)
			}
			proposed = nil
			report.Tick = decision.Update.Tick
			if options.OnTick != nil {
				if err := options.OnTick(*decision.Update); err != nil {
					return report, fmt.Errorf("tick callback: %w", err)
				}
			}
		case decision.Request != nil:
			request := decision.Request
			robot, ok := robots[request.RobotID]
			if !ok || report.Outcomes[request.RobotID].Status != Stopped {
				return report, fmt.Errorf("%w: unavailable robot %q", ErrProtocol, request.RobotID)
			}
			if resumed[request.RobotID] != (request.ReturnValue != nil) {
				return report, fmt.Errorf("%w: unexpected or missing return value for %q", ErrProtocol, request.RobotID)
			}
			if value := request.ReturnValue; value != nil && value.Kind != game.Bool && value.Kind != game.Integer && value.Kind != game.Unit && value.Kind != game.Field {
				return report, fmt.Errorf("%w: unknown return value type %q", ErrProtocol, value.Kind)
			}
			if report.Resumes == options.MaxResumes {
				return report, ErrResumeLimit
			}
			report.Resumes++
			resumed[request.RobotID] = true
			step, err := robot.Next(ctx, request.ReturnValue)
			if cancelErr := ctx.Err(); cancelErr != nil {
				return report, cancelErr
			}
			if err == nil && (step.Done == (step.Action != nil)) {
				err = fmt.Errorf("%w: robot must return an action or completion", ErrProtocol)
			}
			reply = &game.Reply{RobotID: request.RobotID, Action: step.Action, Done: step.Done, Err: err}
			if err != nil {
				reply.Action, reply.Done = nil, false
				report.Outcomes[request.RobotID] = Outcome{Status: Failed, Err: err}
			} else if step.Done {
				report.Outcomes[request.RobotID] = Outcome{Status: Completed}
			}
		case decision.Done:
			for _, outcome := range report.Outcomes {
				if outcome.Status == Stopped {
					return report, fmt.Errorf("%w: board finished with active robots", ErrProtocol)
				}
			}
			return report, nil
		}
	}
}

func validDecision(d game.Decision) bool {
	count := 0
	for _, set := range []bool{d.Request != nil, d.AdvanceTo != nil, d.Update != nil, d.Done} {
		if set {
			count++
		}
	}
	return count == 1
}
