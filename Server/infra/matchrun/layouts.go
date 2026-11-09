package matchrun

import (
	"context"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

var _ usecase.MatchLayoutRepository = (*Layouts)(nil)

type Layouts struct{}

func NewLayouts() *Layouts { return &Layouts{} }

// List supplies provisional symmetric layouts until battle parameters are fixed.
// Robot starts, territories and scoring stay the same across the candidates.
func (*Layouts) List(ctx context.Context) ([]domain.BattleLayout, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	layouts := make([]domain.BattleLayout, 2)
	for i := range layouts {
		layout := domain.BattleLayout{
			ID: fmt.Sprintf("battle-1-layout-%d", i+1), FieldID: "battle-1",
			DrillLimit: 3, Turns: 200, LuggagePoints: 1, DiscardPenalty: 1,
			Terrain: make([][]domain.BattleCell, 7),
		}
		for y := range layout.Terrain {
			layout.Terrain[y] = make([]domain.BattleCell, 9)
			for x := range layout.Terrain[y] {
				cell := domain.BattleCell{Terrain: "land"}
				switch x {
				case 0:
					cell.TerritoryOwner = 1
				case 8:
					cell.TerritoryOwner = 2
				}
				layout.Terrain[y][x] = cell
			}
		}
		for robot := range domain.BattleRobotCount {
			y := 1 + 2*robot
			layout.Robots[0][robot] = domain.BattleRobot{X: 0, Y: y, Direction: "east"}
			layout.Robots[1][robot] = domain.BattleRobot{X: 8, Y: y, Direction: "west"}
			layout.Terrain[y][4].HasLuggage = true
		}
		for _, y := range []int{2, 4} {
			for _, x := range []int{2 + i, 6 - i} {
				layout.Terrain[y][x].Terrain = "rock"
			}
		}
		layouts[i] = layout
	}
	return layouts, nil
}
