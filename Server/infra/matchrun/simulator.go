// Package matchrun simulates battles in the server process.
package matchrun

import (
	"context"
	"crypto/rand"
	"encoding/binary"
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
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

// Simulator adapts Simulate to usecase.MatchSimulator.
type Simulator struct{}

var _ usecase.MatchSimulator = Simulator{}

func (Simulator) Simulate(ctx context.Context, player, opponent []domain.Program, layout domain.BattleLayout) (domain.MatchResult, error) {
	return Simulate(ctx, player, opponent, layout)
}

// Simulate runs both teams on layout and returns the scores and the encoded replay.
func Simulate(ctx context.Context, player, opponent []domain.Program, layout domain.BattleLayout) (domain.MatchResult, error) {
	if err := ctx.Err(); err != nil {
		return domain.MatchResult{}, err
	}
	if len(player) != domain.BattleRobotCount || len(opponent) != domain.BattleRobotCount {
		return domain.MatchResult{}, domain.ErrInvalidProgram
	}
	limits := interpreter.Limits{MaxSteps: 10_000, MaxMemoryBytes: 1 << 20, MaxCallDepth: 64}
	var participants []hub.Participant
	states := make(map[protocol.RobotID]board.RobotState, 2*domain.BattleRobotCount)
	for team, programs := range [][]domain.Program{player, opponent} {
		for slot, program := range programs {
			var seed [8]byte
			if _, err := rand.Read(seed[:]); err != nil {
				return domain.MatchResult{}, fmt.Errorf("generate battle seed: %w", err)
			}
			bot, err := robot.NewPicolangBot(program.Source, binary.LittleEndian.Uint64(seed[:]), limits, false)
			id := protocol.RobotID(fmt.Sprintf("team-%d-robot-%d", team+1, slot+1))
			participant := hub.Participant{ID: id}
			if err != nil {
				if team == 0 {
					return domain.MatchResult{}, fmt.Errorf("%w: robot %d: %w", domain.ErrInvalidProgram, slot+1, err)
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
			states[id] = board.NewRobotState(board.NewPosition(start.Position.X, start.Position.Y), board.Direction(start.Direction), layout.DrillLimit).WithTeam(board.TeamID(team + 1))
		}
	}
	b, err := newBoard(layout, states)
	if err != nil {
		return domain.MatchResult{}, fmt.Errorf("build battle board: %w", err)
	}
	game, err := hub.New(b, participants)
	if err != nil {
		return domain.MatchResult{}, fmt.Errorf("build battle hub: %w", err)
	}
	recording := contextRecorder{ctx: ctx}
	// Queries can resume a robot repeatedly without advancing a turn. Every
	// action charges at least one VM step; allow a final resume to stop each VM.
	report, err := game.Run(hub.Options{MaxResumes: uint64(len(participants)) * (limits.MaxSteps + 1), Recorder: &recording})
	if err != nil {
		return domain.MatchResult{}, fmt.Errorf("run battle simulation: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return domain.MatchResult{}, err
	}
	outcome, ok := report.Result().(boardmsg.BattleResult)
	if !ok {
		return domain.MatchResult{}, fmt.Errorf("unexpected battle result %T", report.Result())
	}
	history, err := replay.History(recording.Events)
	if err != nil {
		return domain.MatchResult{}, err
	}
	result := api.MatchResult{History: history}
	result.Summary.Score.Player, result.Summary.Score.Opponent = outcome.Team1().Score(), outcome.Team2().Score()
	encoded, err := json.Marshal(result)
	if err != nil {
		return domain.MatchResult{}, fmt.Errorf("encode match result: %w", err)
	}
	return domain.MatchResult{ChallengerScore: outcome.Team1().Score(), OpponentScore: outcome.Team2().Score(), Replay: encoded}, nil
}

func newBoard(layout domain.BattleLayout, robots map[protocol.RobotID]board.RobotState) (*board.Board, error) {
	terrain := make([][]board.Terrain, len(layout.Terrain))
	for y, row := range layout.Terrain {
		terrain[y] = make([]board.Terrain, len(row))
		for x, cell := range row {
			switch cell {
			case schema.TerrainLand:
				terrain[y][x] = board.NewLandTerrain(board.PlainLand{}, false, board.NoTeam)
			case schema.TerrainRock:
				terrain[y][x] = board.RockTerrain{}
			case schema.TerrainSea:
				terrain[y][x] = board.SeaTerrain{}
			default:
				return nil, fmt.Errorf("invalid battle terrain %q", cell)
			}
		}
	}
	for team, positions := range layout.Territories {
		for _, position := range positions {
			land, err := landAt(terrain, position)
			if err != nil {
				return nil, err
			}
			if land.TerritoryOwner() != board.NoTeam {
				return nil, fmt.Errorf("overlapping battle territories at %+v", position)
			}
			terrain[position.Y][position.X] = board.NewLandTerrain(board.PlainLand{}, false, board.TeamID(team+1))
		}
	}
	for _, position := range layout.Luggage {
		land, err := landAt(terrain, position)
		if err != nil {
			return nil, err
		}
		if land.HasLuggage() {
			return nil, fmt.Errorf("overlapping battle luggage at %+v", position)
		}
		terrain[position.Y][position.X] = board.NewLandTerrain(board.PlainLand{}, true, land.TerritoryOwner())
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

func landAt(terrain [][]board.Terrain, position schema.Position) (board.LandTerrain, error) {
	if position.Y < 0 || position.Y >= len(terrain) || position.X < 0 || position.X >= len(terrain[position.Y]) {
		return board.LandTerrain{}, fmt.Errorf("battle position %+v is outside the field", position)
	}
	land, ok := terrain[position.Y][position.X].(board.LandTerrain)
	if !ok {
		return board.LandTerrain{}, fmt.Errorf("battle position %+v must be on land", position)
	}
	return land, nil
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
