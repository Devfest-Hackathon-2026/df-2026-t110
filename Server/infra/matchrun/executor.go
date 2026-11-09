// Package matchrun executes battle simulations in the server process.
package matchrun

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	"github.com/dvpico/Programming-Learning-AR/Server/game/recorder"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/replay"
	wire "github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/history"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

var _ usecase.MatchExecutor = (*Executor)(nil)

type Executor struct{}

func NewExecutor() *Executor { return &Executor{} }

func (*Executor) Execute(ctx context.Context, player, opponent []domain.Program, layout domain.BattleLayout) (jsontext.Value, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(player) != domain.BattleRobotCount || len(opponent) != domain.BattleRobotCount {
		return nil, domain.ErrInvalidProgram
	}
	var participants []hub.Participant
	states := make(map[protocol.RobotID]board.RobotState, 2*domain.BattleRobotCount)
	for team, programs := range [][]domain.Program{player, opponent} {
		for slot, program := range programs {
			var seed [8]byte
			if _, err := rand.Read(seed[:]); err != nil {
				return nil, fmt.Errorf("generate battle seed: %w", err)
			}
			bot, err := robot.NewPicolangBot(program.Source, binary.LittleEndian.Uint64(seed[:]), interpreter.Limits{
				MaxSteps: 10_000, MaxMemoryBytes: 1 << 20, MaxCallDepth: 64,
			}, false)
			id := protocol.RobotID(fmt.Sprintf("team-%d-robot-%d", team+1, slot+1))
			participant := hub.Participant{ID: id}
			if err != nil {
				if team == 0 {
					return nil, fmt.Errorf("%w: robot %d: %w", domain.ErrInvalidProgram, slot+1, err)
				}
				// A broken defense program stops its robot, without blocking all
				// challengers who happen to draw this opponent.
				participant.Robot = failedRobot{err: err}
			} else {
				defer func() { _ = bot.Close() }()
				participant.Robot = bot
			}
			participants = append(participants, participant)
			start := layout.Robots[team][slot]
			states[id] = board.NewRobotState(board.NewPosition(start.X, start.Y), board.Direction(start.Direction), layout.DrillLimit).WithTeam(board.TeamID(team + 1))
		}
	}
	b, err := newBoard(layout, states)
	if err != nil {
		return nil, fmt.Errorf("build battle board: %w", err)
	}
	game, err := hub.New(b, participants)
	if err != nil {
		return nil, fmt.Errorf("build battle hub: %w", err)
	}
	recording := contextRecorder{ctx: ctx}
	// Every robot starts once and resumes at most once per turn.
	report, err := game.Run(hub.Options{MaxResumes: 2 * domain.BattleRobotCount * (layout.Turns + 1), Recorder: &recording})
	if err != nil {
		return nil, fmt.Errorf("run battle simulation: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	outcome, ok := report.Result().(boardmsg.BattleResult)
	if !ok {
		return nil, fmt.Errorf("unexpected battle result %T", report.Result())
	}
	history, err := replay.History(recording.Events)
	if err != nil {
		return nil, err
	}
	var result struct {
		History wire.GameHistory `json:"history"`
		Summary struct {
			Score struct {
				Player   int64 `json:"player"`
				Opponent int64 `json:"opponent"`
			} `json:"score"`
		} `json:"summary"`
	}
	result.History = history
	result.Summary.Score.Player, result.Summary.Score.Opponent = outcome.Team1().Score(), outcome.Team2().Score()
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("encode match result: %w", err)
	}
	return encoded, nil
}

func newBoard(layout domain.BattleLayout, robots map[protocol.RobotID]board.RobotState) (*board.Board, error) {
	terrain := make([][]board.Terrain, len(layout.Terrain))
	for y, row := range layout.Terrain {
		terrain[y] = make([]board.Terrain, len(row))
		for x, cell := range row {
			switch cell.Terrain {
			case "land":
				terrain[y][x] = board.NewLandTerrain(board.PlainLand{}, cell.HasLuggage, board.TeamID(cell.TerritoryOwner))
			case "rock":
				terrain[y][x] = board.RockTerrain{}
			case "sea":
				terrain[y][x] = board.SeaTerrain{}
			default:
				return nil, fmt.Errorf("invalid battle terrain %q", cell.Terrain)
			}
		}
	}
	rules, err := board.NewBattleConfig(layout.Turns, layout.LuggagePoints, layout.DiscardPenalty).CompletionRules()
	if err != nil {
		return nil, err
	}
	config, err := board.NewConfig(terrain, robots, 0, rules)
	if err != nil {
		return nil, err
	}
	return board.New(config)
}

type contextRecorder struct {
	recorder.Recorder
	ctx context.Context
}

func (r *contextRecorder) Record(event gameevent.GameEvent) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	return r.Recorder.Record(event)
}

type failedRobot struct{ err error }

func (r failedRobot) Start() (robotmsg.RobotOutput, error) { return robotmsg.RobotOutput{}, r.err }
func (r failedRobot) Next(values.Value) (robotmsg.RobotOutput, error) {
	return robotmsg.RobotOutput{}, r.err
}
