package function

import (
	"fmt"
	"slices"

	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
)

func matches(store *memory.Store, id memory.ID, t bot.Type) bool {
	value, exists := store.Get(id)
	if !exists {
		return false
	}
	switch bot.NodeValue(t).(type) {
	case bot.BoolType:
		_, ok := value.(memory.Bool)
		return ok
	case bot.IntegerType:
		_, ok := value.(memory.Integer)
		return ok
	case bot.UnitType:
		_, ok := value.(memory.Unit)
		return ok
	case bot.TerrainType:
		_, ok := value.(memory.Terrain)
		return ok
	case bot.RotationType:
		_, ok := value.(memory.Rotation)
		return ok
	case bot.DistanceTargetType:
		_, ok := value.(memory.DistanceTarget)
		return ok
	case bot.StackType:
		switch value.(type) {
		case memory.ListRoot, memory.ListNode:
			return true
		}
	}
	return false
}

func (m *Machine) define(ident bot.Ident[bot.VariableIndex], id memory.ID) {
	if ident.Variable != bot.VariableIndex(len(m.variables)) {
		panic(fmt.Errorf("invalid declaration slot %d", ident.Variable))
	}
	m.resources.Time.Increment()
	if m.resources.Events != nil {
		m.resources.Events.Add(robotevent.VariableInitEvent{HintID: ident.HintID, Value: m.record(id)})
	}
	m.variables = append(m.variables, binding{value: id, hint: ident.HintID, announced: m.resources.Events != nil})
}

func (m *Machine) assign(index bot.VariableIndex, id memory.ID) {
	variable := &m.variables[index]
	m.resources.Time.Increment()
	if m.resources.Events != nil {
		diff := m.replacement(id)
		m.resources.Events.Add(robotevent.VariableUpdateEvent{HintID: variable.hint, Diff: []robotevent.ValueUpdate{diff}})
	}
	previous := variable.value
	variable.value = id
	m.drop(previous)
}

// Snapshots own their stack slices. Subsequent assignments or memory releases
// cannot change values already emitted to the event collector.
func (m *Machine) record(id memory.ID) values.RecordValue {
	switch v := m.value(id).(type) {
	case memory.Bool:
		return values.BoolValue{Bool: v.Value}
	case memory.Integer:
		return values.IntegerValue{Integer: v.Value}
	case memory.Unit:
		return values.UnitValue{}
	case memory.Terrain:
		return v.Value
	case memory.Rotation:
		return values.RotationRecordValue{Rotation: v.Value}
	case memory.DistanceTarget:
		return v.Value
	case memory.ListRoot:
		return values.StackRecordValue{}
	case memory.ListNode:
		var elements []values.RecordValue
		for {
			elements = append(elements, m.record(v.Value))
			switch prev := m.value(v.Prev).(type) {
			case memory.ListRoot:
				slices.Reverse(elements)
				return values.StackRecordValue{Elements: elements}
			case memory.ListNode:
				v = prev
			default:
				panic(fmt.Errorf("list previous value has kind %T", prev))
			}
		}
	default:
		panic(fmt.Errorf("cannot record value %T", v))
	}
}

// Full replacements also cover assignments from another variable or from
// get_back; choosing a smaller stack diff can be added without changing values.
func (m *Machine) replacement(id memory.ID) robotevent.ValueUpdate {
	switch v := m.record(id).(type) {
	case values.BoolValue:
		return robotevent.BoolUpdate{Value: v.Bool}
	case values.IntegerValue:
		return robotevent.IntegerUpdate{Value: v.Integer}
	case values.UnitValue:
		return robotevent.UnitUpdate{}
	case values.TerrainValue:
		return robotevent.TerrainUpdate{Value: v}
	case values.RotationRecordValue:
		return robotevent.RotationUpdate{Value: v.Rotation}
	case values.DistanceTarget:
		return robotevent.DistanceTargetUpdate{Value: v}
	case values.StackRecordValue:
		return robotevent.StackUpdate{Update: robotevent.StackReplace{Value: v.Elements}}
	default:
		panic(fmt.Errorf("cannot replace recorded value %T", v))
	}
}
