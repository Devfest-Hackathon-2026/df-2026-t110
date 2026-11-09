package function

import (
	"fmt"

	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
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
		m.resources.Events.Add(robotevent.VariableInitEvent{HintID: ident.HintID, ValueID: id})
	}
	m.variables = append(m.variables, binding{value: id, hint: ident.HintID, announced: m.resources.Events != nil})
}

func (m *Machine) assign(index bot.VariableIndex, id memory.ID) {
	variable := &m.variables[index]
	m.resources.Time.Increment()
	if m.resources.Events != nil {
		m.resources.Events.Add(robotevent.VariableUpdateEvent{HintID: variable.hint, ValueID: id})
	}
	previous := variable.value
	variable.value = id
	m.drop(previous)
}
