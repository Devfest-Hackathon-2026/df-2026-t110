// Package tutorialrun executes tutorial simulations in the request process.
package tutorialrun

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"math"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	"github.com/dvpico/Programming-Learning-AR/Server/game/recorder"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter"
	"github.com/dvpico/Programming-Learning-AR/Server/game/tutorial"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/replay"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// A goal or turn limit can stop the board while the VM is suspended.
	// Record its cleanup and final status before serializing the history.
	if err := bot.Close(); err != nil {
		return nil, fmt.Errorf("close tutorial VM: %w", err)
	}
	for _, event := range bot.Collect() {
		recording.Events = append(recording.Events, gameevent.RobotEvent{InterpreterID: robotID, Event: event})
	}
	outcome, ok := report.Result().(boardmsg.TutorialResult)
	if !ok {
		return nil, fmt.Errorf("unexpected tutorial result %T", report.Result())
	}
	history, err := replay.History(recording.Events)
	if err != nil {
		return nil, err
	}
	result := api.TutorialResult{History: history}
	result.Summary.Cleared = outcome.Cleared()
	tick := outcome.Tick()
	if tick > uint64(math.MaxInt) {
		return nil, fmt.Errorf("tutorial tick exceeds integer range")
	}
	result.Summary.Tick = int(tick)
	result.Summary.Reason = api.TutorialResultSummaryReason(outcome.EndReason())
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("encode tutorial result: %w", err)
	}
	return encoded, nil
}
