package builtin

import (
	"errors"
	"fmt"

	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
)

func (e *Evaluator) evaluate(name bot.Name, ids []memory.ID, args []memory.Value) (memory.ID, error) {
	integer := func(i int) int64 { return args[i].(memory.Integer).Value }
	boolean := func(i int) bool { return args[i].(memory.Bool).Value }
	var result memory.Value
	switch name {
	case "not":
		result = memory.Bool{Value: !boolean(0)}
	case "and", "or", "bool_eq", "bool_neq":
		left, right := boolean(0), boolean(1)
		switch name {
		case "and":
			result = memory.Bool{Value: left && right}
		case "or":
			result = memory.Bool{Value: left || right}
		case "bool_eq":
			result = memory.Bool{Value: left == right}
		case "bool_neq":
			result = memory.Bool{Value: left != right}
		}
	case "add":
		result = memory.Integer{Value: integer(0) + integer(1)}
	case "minus":
		result = memory.Integer{Value: -integer(0)}
	case "sub":
		result = memory.Integer{Value: integer(0) - integer(1)}
	case "mul":
		result = memory.Integer{Value: integer(0) * integer(1)}
	case "div":
		if integer(1) == 0 {
			return 0, ErrDivisionByZero
		}
		result = memory.Integer{Value: integer(0) / integer(1)}
	case "integer_eq":
		result = memory.Bool{Value: integer(0) == integer(1)}
	case "integer_neq":
		result = memory.Bool{Value: integer(0) != integer(1)}
	case "gt":
		result = memory.Bool{Value: integer(0) > integer(1)}
	case "lt":
		result = memory.Bool{Value: integer(0) < integer(1)}
	case "geq":
		result = memory.Bool{Value: integer(0) >= integer(1)}
	case "leq":
		result = memory.Bool{Value: integer(0) <= integer(1)}
	case "random_bool":
		result = memory.Bool{Value: e.random.Uint64()&1 == 1}
	case "random_integer":
		first, last := integer(0), integer(1)
		if last < first {
			return 0, errors.New("random_integer upper bound is smaller than lower bound")
		}
		// Unsigned arithmetic covers inclusive intervals crossing zero and the
		// full int64 domain. Only that full-domain interval has width zero.
		width := uint64(last) - uint64(first) + 1 //nolint:gosec // Interval arithmetic intentionally wraps modulo 2^64 to cover signed ranges.
		if width == 0 {
			result = memory.Integer{Value: int64(e.random.Uint64())} //nolint:gosec // All 64 random bits intentionally represent the full signed int64 domain.
		} else {
			result = memory.Integer{Value: int64(uint64(first) + e.random.Uint64N(width))} //nolint:gosec // Modulo arithmetic maps the unsigned offset into the signed interval.
		}
	case "new":
		result = memory.ListRoot{}
	case "some":
		result = memory.Some{Element: ids[0]}
	case "none":
		result = memory.None{}
	case "unwrap":
		some, ok := args[0].(memory.Some)
		if !ok {
			return 0, ErrUnwrapNone
		}
		return some.Element, e.memory.Retain(some.Element)
	case "is_some":
		_, present := args[0].(memory.Some)
		result = memory.Bool{Value: present}
	case "push_back":
		result = memory.ListNode{Prev: ids[0], Value: ids[1]}
	case "pop_back", "get_back":
		node, ok := args[0].(memory.ListNode)
		if !ok {
			return 0, fmt.Errorf("%s requires a nonempty stack", name)
		}
		id := node.Prev
		if name == "get_back" {
			id = node.Value
		}
		return id, e.memory.Retain(id)
	case "size":
		id, count := ids[0], int64(0)
		for {
			value, _ := e.memory.Get(id)
			node, ok := value.(memory.ListNode)
			if !ok {
				break
			}
			count++
			id = node.Prev
		}
		result = memory.Integer{Value: count}
	case "is_land", "is_goal", "is_own_territory", "is_enemy_territory", "is_checkpoint", "is_luggage", "is_sea", "is_rock":
		terrain := args[0].(memory.Terrain).Value
		var match bool
		switch name {
		case "is_land":
			_, match = terrain.(values.LandTerrain)
		case "is_sea":
			_, match = terrain.(values.SeaTerrain)
		case "is_rock":
			_, match = terrain.(values.RockTerrain)
		default:
			if land, ok := terrain.(values.LandTerrain); ok {
				switch name {
				case "is_goal":
					_, match = land.Kind.(values.GoalLand)
				case "is_own_territory":
					match = land.Territory == values.OwnTerritory
				case "is_enemy_territory":
					match = land.Territory == values.EnemyTerritory
				case "is_checkpoint":
					_, match = land.Kind.(values.CheckpointLand)
				case "is_luggage":
					match = land.HasLuggage
				}
			}
		}
		result = memory.Bool{Value: match}
	default:
		return 0, fmt.Errorf("builtin %q has no evaluator", name)
	}
	return e.memory.Add(result)
}
