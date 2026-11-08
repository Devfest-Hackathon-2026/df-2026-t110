package interpreter

import (
	"testing"

	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
)

func TestPrimitivePrintsEvaluateArgumentsAndResumeWithUnit(t *testing.T) {
	t.Parallel()
	b := &builder{}
	rotation := func() expr {
		return bot.LiteralExpr[bot.Name, bot.Untyped]{Source: b.source(), Value: bot.RotationLiteral{Value: values.Left}}
	}
	main := b.function(nil, bot.UnitType{},
		b.call("print_unit", b.call("rotate", rotation())),
		b.call("print_terrain", b.call("search_front")),
		b.call("print_rotation", rotation()),
		b.call("print_distance_target", b.target(values.DistanceSea)), b.ret(b.unit()))
	v := newVM(t, checked(t, map[bot.Name]fn{"main": main}), 0)
	base := snapshot(v).MemoryBytes
	output, err := v.Start()
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.RotateAction{Rotation: values.Left}})
	output, err = v.Next(values.UnitValue{})
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.PrintUnitAction{}})
	output, err = v.Next(values.UnitValue{})
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.SearchFrontAction{}})
	terrain := values.LandTerrain{Kind: values.PlainLand{}, HasLuggage: true, Territory: values.EnemyTerritory}
	output, err = v.Next(terrain)
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.PrintTerrainAction{Terrain: terrain}})
	output, err = v.Next(values.UnitValue{})
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.PrintRotationAction{Rotation: values.Left}})
	output, err = v.Next(values.UnitValue{})
	expectOutput(t, output, err, robotmsg.ActionStep{Action: actions.PrintDistanceTargetAction{DistanceTarget: values.DistanceSea}})
	output, err = v.Next(values.UnitValue{})
	expectOutput(t, output, err, robotmsg.DoneStep{})
	assertClean(t, v, base)
}
