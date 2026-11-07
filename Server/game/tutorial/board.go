// Package tutorial builds a playable board from a tutorial stage definition.
package tutorial

import (
	"fmt"
	"slices"

	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
)

// NewBoard builds the selected zero-based variant and its completion rules.
// Use robotID when registering the tutorial robot with the Hub.
func NewBoard(stage schema.StageDefinition, variant int, robotID protocol.RobotID, maxTicks uint64) (*board.Board, error) {
	if variant < 0 || variant >= len(stage.Variants) {
		return nil, fmt.Errorf("tutorial variant %d is out of range", variant)
	}
	field := stage.Variants[variant]
	terrain, err := buildTerrain(field)
	if err != nil {
		return nil, err
	}
	var checkpoints []int64
	orders := make(map[int64]struct{})
	for _, object := range field.Objects {
		if err := placeObject(terrain, object); err != nil {
			return nil, err
		}
		if object.Checkpoint != nil {
			order := int64(object.Checkpoint.Order)
			if _, duplicate := orders[order]; duplicate {
				return nil, fmt.Errorf("duplicate checkpoint order %d", order)
			}
			orders[order] = struct{}{}
			checkpoints = append(checkpoints, order)
		}
	}
	slices.Sort(checkpoints)

	rules, err := newCompletionRules(stage.ClearCondition, robotID, checkpoints)
	if err != nil {
		return nil, err
	}
	position := field.Robot.Position
	robot := board.NewRobotState(board.NewPosition(position.X, position.Y), board.Direction(field.Robot.Direction), int64(field.DrillLimit))
	config, err := board.NewConfig(terrain, map[protocol.RobotID]board.RobotState{robotID: robot}, maxTicks, rules)
	if err != nil {
		return nil, err
	}

	return board.New(config)
}

func buildTerrain(field schema.FieldDefinition) ([][]board.Terrain, error) {
	if field.Width <= 0 || field.Height <= 0 || len(field.Terrain) != field.Height {
		return nil, fmt.Errorf("invalid tutorial field dimensions %dx%d", field.Width, field.Height)
	}

	terrain := make([][]board.Terrain, field.Height)
	for y, row := range field.Terrain {
		if len(row) != field.Width {
			return nil, fmt.Errorf("tutorial terrain row %d has width %d, want %d", y, len(row), field.Width)
		}

		terrain[y] = make([]board.Terrain, field.Width)
		for x, cell := range row {
			switch cell {
			case schema.TerrainLand:
				terrain[y][x] = board.NewLandTerrain(board.PlainLand{}, false, board.NoTeam)
			case schema.TerrainRock:
				terrain[y][x] = board.RockTerrain{}
			case schema.TerrainSea:
				terrain[y][x] = board.SeaTerrain{}
			default:
				return nil, fmt.Errorf("invalid tutorial terrain %q at (%d, %d)", cell, x, y)
			}
		}
	}

	return terrain, nil
}

func placeObject(terrain [][]board.Terrain, object schema.FieldObject) error {
	var position schema.Position
	var kind board.LandKind
	var luggage bool
	switch {
	case object.Goal != nil && object.Checkpoint == nil && object.Luggage == nil:
		position, kind = object.Goal.Position, board.GoalLand{}
	case object.Checkpoint != nil && object.Goal == nil && object.Luggage == nil:
		position, kind = object.Checkpoint.Position, board.NewCheckpointLand(int64(object.Checkpoint.Order))
	case object.Luggage != nil && object.Goal == nil && object.Checkpoint == nil:
		position, luggage = object.Luggage.Position, true
	default:
		return fmt.Errorf("tutorial field object must have exactly one variant")
	}

	x, y := position.X, position.Y
	if y < 0 || y >= len(terrain) || x < 0 || x >= len(terrain[y]) {
		return fmt.Errorf("tutorial object at (%d, %d) is outside the field", x, y)
	}
	land, ok := terrain[y][x].(board.LandTerrain)
	if !ok {
		return fmt.Errorf("tutorial object at (%d, %d) must be on land", x, y)
	}
	if luggage {
		if land.HasLuggage() {
			return fmt.Errorf("multiple pieces of luggage at (%d, %d)", x, y)
		}
		kind = land.Kind()
	} else if _, plain := land.Kind().(board.PlainLand); !plain {
		return fmt.Errorf("overlapping goal or checkpoint at (%d, %d)", x, y)
	}
	terrain[y][x] = board.NewLandTerrain(kind, luggage || land.HasLuggage(), board.NoTeam)
	return nil
}
