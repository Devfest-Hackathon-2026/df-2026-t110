package matchrun

import (
	"context"
	_ "embed"
	json "encoding/json/v2"
	"fmt"
	"slices"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

var _ usecase.MatchLayoutRepository = (*Layouts)(nil)

//go:embed layouts.json
var layoutJSON []byte

type Layouts struct{ layouts []domain.BattleLayout }

func NewLayouts() (*Layouts, error) {
	var layouts []domain.BattleLayout
	if err := json.Unmarshal(layoutJSON, &layouts, json.RejectUnknownMembers(true)); err != nil {
		return nil, fmt.Errorf("decode battle layouts: %w", err)
	}
	return &Layouts{layouts: layouts}, nil
}

// List copies the cached definitions so callers never share mutable board data.
func (r *Layouts) List(ctx context.Context) ([]domain.BattleLayout, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	layouts := slices.Clone(r.layouts)
	for i := range layouts {
		layout := &layouts[i]
		layout.Terrain = slices.Clone(layout.Terrain)
		for y := range layout.Terrain {
			layout.Terrain[y] = slices.Clone(layout.Terrain[y])
		}
		layout.Luggage = slices.Clone(layout.Luggage)
		for team := range layout.Territories {
			layout.Territories[team] = slices.Clone(layout.Territories[team])
		}
	}
	return layouts, nil
}
