package main

import (
	"context"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/game"
	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
)

// battleOptions controls reproducible simulations and streaming recording.
type battleOptions struct {
	Seeds    [2][domain.BattleRobotCount]uint64
	Limits   interpreter.Limits
	Recorder game.Recorder
}

func defaultLimits() interpreter.Limits {
	return interpreter.Limits{MaxSteps: 10_000, MaxMemoryBytes: 1 << 20, MaxCallDepth: 64}
}

// simulate wires the CLI's programs and layout into the existing board and hub.
// Turn progression, scoring and event delivery are owned by those components.
func simulate(ctx context.Context, player, opponent []domain.Program, layout domain.BattleLayout, options battleOptions) (hub.Report, error) {
	if err := ctx.Err(); err != nil {
		return hub.Report{}, err
	}
	if len(player) != domain.BattleRobotCount || len(opponent) != domain.BattleRobotCount {
		return hub.Report{}, domain.ErrInvalidProgram
	}
	limits := options.Limits
	if limits.MaxSteps == 0 || limits.MaxSteps > (^uint64(0)/uint64(2*domain.BattleRobotCount))-1 || limits.MaxMemoryBytes == 0 || limits.MaxCallDepth == 0 {
		return hub.Report{}, fmt.Errorf("simulation limits must be positive and resumes must fit uint64")
	}
	var participants []hub.Participant
	states := make(map[protocol.RobotID]board.RobotState, 2*domain.BattleRobotCount)
	for team, programs := range [][]domain.Program{player, opponent} {
		for slot, program := range programs {
			bot, err := robot.NewPicolangBot(program.Source, options.Seeds[team][slot], limits, false)
			id := protocol.RobotID(fmt.Sprintf("team-%d-robot-%d", team+1, slot+1))
			participant := hub.Participant{ID: id}
			if err != nil {
				return hub.Report{}, fmt.Errorf("%w: team %d robot %d: %w", domain.ErrInvalidProgram, team+1, slot+1, err)
			}
			defer func() { _ = bot.Close() }()
			participant.Robot = bot
			participants = append(participants, participant)
			start := layout.Robots[team][slot]
			states[id] = board.NewRobotState(board.NewPosition(start.Position.X, start.Position.Y), board.Direction(start.Direction), layout.DrillLimit).WithTeam(board.TeamID(team + 1))
		}
	}
	b, err := newBoard(layout, states)
	if err != nil {
		return hub.Report{}, fmt.Errorf("build battle board: %w", err)
	}
	game, err := hub.New(b, participants)
	if err != nil {
		return hub.Report{}, fmt.Errorf("build battle hub: %w", err)
	}
	// Queries can resume a robot repeatedly without advancing a turn. Every
	// action charges at least one VM step; allow a final resume to stop each VM.
	report, err := game.Run(hub.Options{MaxResumes: uint64(len(participants)) * (limits.MaxSteps + 1), Recorder: options.Recorder})
	if err != nil {
		return report, fmt.Errorf("run battle simulation: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	_, ok := report.Result().(boardmsg.BattleResult)
	if !ok {
		return report, fmt.Errorf("unexpected battle result %T", report.Result())
	}
	return report, nil
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
