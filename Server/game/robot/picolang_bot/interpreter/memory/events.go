package memory

import payload "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/memory"

// Allocation payloads contain scalar data and IDs only. Converting one stored
// node never traverses a stack or copies its elements.
func allocationValue(value Value) payload.Value {
	switch v := value.(type) {
	case Bool:
		return payload.Bool{Bool: v.Value}
	case Integer:
		return payload.Integer{Integer: v.Value}
	case Unit:
		return payload.Unit{}
	case Terrain:
		return payload.Terrain{Terrain: v.Value}
	case Rotation:
		return payload.Rotation{Rotation: v.Value}
	case DistanceTarget:
		return payload.DistanceTarget{Target: v.Value}
	case ListNode:
		return payload.ListNode{Prev: v.Prev, Element: v.Value}
	case Some:
		return payload.Some{Element: v.Element}
	case None:
		return payload.None{}
	default:
		panic("recording unsupported memory allocation")
	}
}
