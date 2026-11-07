package engine

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func wait(id protocol.RobotID, ticks int64) fieldevent.WaitResult {
	return fieldevent.WaitResult{RobotID: id, Ticks: ticks}
}

func (b *Board) advanceWaits(ticks uint64) {
	for id, state := range b.controls {
		waiting, ok := state.(waitingControl)
		if !ok {
			continue
		}
		if waiting.remaining <= ticks {
			b.controls[id] = readyControl{value: values.UnitValue{}}
			continue
		}
		waiting.remaining -= ticks
		b.controls[id] = waiting
	}
}
