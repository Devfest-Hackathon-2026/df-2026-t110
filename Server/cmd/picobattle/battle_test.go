package main

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	"github.com/dvpico/Programming-Learning-AR/Server/game/recorder"
)

type recordingFunc func(gameevent.GameEvent) error

func (f recordingFunc) Record(event gameevent.GameEvent) error { return f(event) }

func TestSimulateStreamsAndStopsOnRecordingFailure(t *testing.T) {
	t.Parallel()
	program, err := generateBot("courier", 1)
	if err != nil {
		t.Fatal(err)
	}
	layouts, err := loadLayouts(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var seen int
	opts := battleOptions{Limits: defaultLimits(), Recorder: recordingFunc(func(gameevent.GameEvent) error {
		seen++
		return io.ErrClosedPipe
	})}
	_, err = simulate(t.Context(), team(program.Source), team(program.Source), layouts[0], opts)
	if !errors.Is(err, io.ErrClosedPipe) || !errors.Is(err, hub.ErrRecording) || seen != 1 {
		t.Errorf("seen=%d err=%v", seen, err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	opts.Recorder = recordingFunc(func(gameevent.GameEvent) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		cancel()
		return nil
	})
	_, err = simulate(ctx, team(program.Source), team(program.Source), layouts[0], opts)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("canceled simulation err=%v", err)
	}
}

func TestSimulateStrictProgramsAndLimits(t *testing.T) {
	t.Parallel()
	program, err := generateBot("courier", 1)
	if err != nil {
		t.Fatal(err)
	}
	layouts, err := loadLayouts(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	opts := battleOptions{Limits: defaultLimits()}
	_, err = simulate(t.Context(), team(program.Source), team([]byte(`{}`)), layouts[0], opts)
	if err == nil {
		t.Error("strict simulation accepted invalid opponent")
	}
	opts.Limits.MaxSteps = ^uint64(0)
	if _, err := simulate(t.Context(), team(program.Source), team(program.Source), layouts[0], opts); err == nil {
		t.Error("accepted overflowing resumes")
	}
	opts.Limits = defaultLimits()
	opts.Limits.MaxCallDepth = 0
	if _, err := simulate(t.Context(), team(program.Source), team(program.Source), layouts[0], opts); err == nil {
		t.Error("accepted zero call depth")
	}
	// The existing hub emits the initial board and turn updates to its recorder.
	opts.Limits = defaultLimits()
	var recording recorder.Recorder
	opts.Recorder = &recording
	if _, err := simulate(t.Context(), team(program.Source), team(program.Source), layouts[0], opts); err != nil {
		t.Error(err)
	}
	if len(recording.Events) < 2 {
		t.Error("missing streamed initial state and update")
	}
}
