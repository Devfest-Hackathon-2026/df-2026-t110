package matchrun

import (
	"context"
	_ "embed"
	json "encoding/json/v2"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

var _ usecase.MatchLayoutRepository = (*Layouts)(nil)

//go:embed layouts.json
var layoutJSON []byte

type Layouts struct{}

func NewLayouts() *Layouts { return &Layouts{} }

// List decodes fresh layouts so callers never share mutable board data.
func (*Layouts) List(ctx context.Context) ([]domain.BattleLayout, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var layouts []domain.BattleLayout
	if err := json.Unmarshal(layoutJSON, &layouts, json.RejectUnknownMembers(true)); err != nil {
		return nil, fmt.Errorf("decode battle layouts: %w", err)
	}
	return layouts, nil
}
