// Package tutorialrun executes tutorial simulations in the request process.
package tutorialrun

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	"github.com/dvpico/Programming-Learning-AR/Server/game/recorder"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter"
	"github.com/dvpico/Programming-Learning-AR/Server/game/tutorial"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

const robotID = "robot-1"

// Executor is stateless; recordings live only for the duration of Execute.
type Executor struct{}

var _ usecase.TutorialExecutor = (*Executor)(nil)

func NewExecutor() *Executor { return &Executor{} }

func (*Executor) Execute(ctx context.Context, program domain.Program, lesson domain.Tutorial, variant int) (jsontext.Value, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var seed [8]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return nil, fmt.Errorf("generate simulation seed: %w", err)
	}
	bot, err := robot.NewPicolangBot(program.Source, binary.LittleEndian.Uint64(seed[:]), interpreter.Limits{
		MaxSteps: 10_000, MaxMemoryBytes: 1 << 20, MaxCallDepth: 64,
	}, true)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrInvalidProgram, err)
	}
	defer func() { _ = bot.Close() }()
	board, err := tutorial.NewBoard(lesson.Stage, variant, robotID, 1_000)
	if err != nil {
		return nil, fmt.Errorf("build tutorial board: %w", err)
	}
	game, err := hub.New(board, []hub.Participant{{ID: robotID, Robot: bot}})
	if err != nil {
		return nil, fmt.Errorf("build tutorial hub: %w", err)
	}
	var recording recorder.Recorder
	report, err := game.Run(hub.Options{MaxResumes: 10_001, Recorder: &recording})
	if err != nil {
		return nil, fmt.Errorf("run tutorial simulation: %w", err)
	}
	// Reaching the goal can interrupt a suspended VM. Preserve its cleanup and
	// execution_finished events along with the events collected by the hub.
	if err := bot.Close(); err != nil {
		return nil, fmt.Errorf("close tutorial robot: %w", err)
	}
	for _, event := range bot.Collect() {
		recording.Events = append(recording.Events, gameevent.RobotEvent{InterpreterID: robotID, Event: event})
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	outcome, ok := report.Result().(boardmsg.TutorialResult)
	if !ok {
		return nil, fmt.Errorf("unexpected tutorial result %T", report.Result())
	}
	history := make([]any, 0, len(recording.Events))
	for _, event := range recording.Events {
		entry, err := historyEvent(event)
		if err != nil {
			return nil, err
		}
		history = append(history, entry)
	}
	result, err := json.Marshal(object{
		"history": history,
		"summary": object{"cleared": outcome.Cleared(), "tick": outcome.Tick(), "reason": outcome.EndReason()},
	})
	if err != nil {
		return nil, fmt.Errorf("encode tutorial result: %w", err)
	}
	return result, nil
}
