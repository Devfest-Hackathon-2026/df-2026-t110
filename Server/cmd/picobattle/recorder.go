package main

import (
	"context"
	json "encoding/json/v2"
	"io"

	"github.com/dvpico/Programming-Learning-AR/Server/game"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/replay"
)

var _ game.Recorder = lineRecorder{}

// lineRecorder converts each event separately, so history is never accumulated.
// Writes go directly to the file, leaving complete earlier lines after a failure.
type lineRecorder struct {
	ctx    context.Context
	writer io.Writer
}

func (r lineRecorder) Record(event gameevent.GameEvent) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	history, err := replay.History([]gameevent.GameEvent{event})
	if err != nil {
		return err
	}
	data, err := json.Marshal(history[0], json.Deterministic(true))
	if err != nil {
		return err
	}
	_, err = r.writer.Write(append(data, '\n'))
	return err
}
