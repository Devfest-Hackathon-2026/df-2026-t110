package action_test

import (
	"testing"

	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func TestPrintActionValidation(t *testing.T) {
	t.Parallel()
	for _, action := range []actions.Action{
		actions.PrintTerrainAction{}, actions.PrintTerrainAction{Terrain: values.LandTerrain{}}, actions.PrintTerrainAction{Terrain: &values.SeaTerrain{}}, actions.PrintTerrainAction{Terrain: values.LandTerrain{Kind: values.GoalLand{}, Territory: values.OwnTerritory}},
		actions.PrintRotationAction{}, actions.PrintRotationAction{Rotation: "north"}, actions.PrintDistanceTargetAction{}, actions.PrintDistanceTargetAction{DistanceTarget: "invalid"}, (*actions.PrintUnitAction)(nil),
	} {
		if actions.ValidAction(action) {
			t.Errorf("invalid print action accepted: %#v", action)
		}
	}
}
