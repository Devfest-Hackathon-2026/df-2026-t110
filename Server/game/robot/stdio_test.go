package robot_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/game"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot"
)

func session(t *testing.T, input io.ReadCloser, output io.Writer) *robot.Session {
	t.Helper()
	s, err := robot.NewSession(input, output)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func stdio(t *testing.T, id game.RobotID, s *robot.Session) *robot.Stdio {
	t.Helper()
	r, err := robot.NewStdio(id, s)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestInteractiveCommandsAndReturnValues(t *testing.T) {
	var output bytes.Buffer
	s := session(t, io.NopCloser(strings.NewReader("\nhelp\np2 forward\np1 rotate sideways\np1 forward extra\np1 drill_durability\np1 rotate left\np1 rotate right\np1 back\np1 forward\np1 done\np2 done\n")), &output)
	p1, p2 := stdio(t, "p1", s), stdio(t, "p2", s)
	tests := []struct {
		value *game.Value
		want  game.Action
	}{
		{nil, game.Action{Kind: game.DrillDurability}},
		{&game.Value{Kind: game.Integer, Integer: 3}, game.Action{Kind: game.Rotate, Turn: game.Left}},
		{&game.Value{Kind: game.Unit}, game.Action{Kind: game.Rotate, Turn: game.Right}},
		{&game.Value{Kind: game.Unit}, game.Action{Kind: game.Back}},
		{&game.Value{Kind: game.Bool, Bool: true}, game.Action{Kind: game.Forward}},
	}
	for _, tt := range tests {
		step, err := p1.Next(t.Context(), tt.value)
		if err != nil || step.Done || step.Action == nil || *step.Action != tt.want {
			t.Fatalf("got %+v, %v; want %+v", step, err, tt.want)
		}
	}
	step, err := p1.Next(t.Context(), &game.Value{Kind: game.Bool, Bool: false})
	if err != nil || !step.Done {
		t.Fatalf("done not accepted: %+v, %v", step, err)
	}
	step, err = p2.Next(t.Context(), nil)
	if err != nil || !step.Done {
		t.Fatalf("shared input not routed: %+v, %v", step, err)
	}
	for _, expected := range []string{"invalid command for p1", "p1 => 3", "p1 => ()", "p1 => true", "p1 => false", "p2> "} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("output missing %q: %s", expected, output.String())
		}
	}
}

func TestEOFAppliesToAllRobots(t *testing.T) {
	s := session(t, io.NopCloser(strings.NewReader("p1 forward")), io.Discard)
	p1, p2 := stdio(t, "p1", s), stdio(t, "p2", s)
	step, err := p1.Next(t.Context(), nil)
	if err != nil || step.Action == nil || step.Action.Kind != game.Forward {
		t.Fatalf("unterminated last line not read: %+v, %v", step, err)
	}
	for _, r := range []*robot.Stdio{p2, p1} {
		step, err := r.Next(t.Context(), nil)
		if err != nil || !step.Done {
			t.Fatalf("EOF not shared: %+v, %v", step, err)
		}
	}
}

type observedInput struct {
	*io.PipeReader
	entered  chan struct{}
	returned chan struct{}
	once     sync.Once
}

func (r *observedInput) Read(p []byte) (int, error) {
	r.once.Do(func() { close(r.entered) })
	n, err := r.PipeReader.Read(p)
	close(r.returned)
	return n, err
}

func TestCancellationUnblocksInput(t *testing.T) {
	reader, writer := io.Pipe()
	t.Cleanup(func() { _ = writer.Close() })
	input := &observedInput{PipeReader: reader, entered: make(chan struct{}), returned: make(chan struct{})}
	s := session(t, input, io.Discard)
	r := stdio(t, "p1", s)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	result := make(chan error, 1)
	go func() { _, err := r.Next(ctx, nil); result <- err }()
	select {
	case <-input.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("input read did not start")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v, want cancellation", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("robot stayed blocked after cancellation")
	}
	select {
	case <-input.returned:
	case <-time.After(2 * time.Second):
		t.Fatal("reader goroutine remained blocked")
	}
}

type failedIO struct{ err error }

func (f failedIO) Read([]byte) (int, error)  { return 0, f.err }
func (f failedIO) Write([]byte) (int, error) { return 0, f.err }
func (f failedIO) Close() error              { return nil }

func TestIOErrorsAndValidation(t *testing.T) {
	failure := errors.New("I/O failed")
	for _, tc := range []struct {
		input  io.ReadCloser
		output io.Writer
	}{
		{failedIO{failure}, io.Discard},
		{io.NopCloser(strings.NewReader("p1 forward\n")), failedIO{failure}},
	} {
		r := stdio(t, "p1", session(t, tc.input, tc.output))
		if _, err := r.Next(t.Context(), nil); !errors.Is(err, failure) {
			t.Fatalf("I/O error lost: %v", err)
		}
	}
	if _, err := robot.NewSession(nil, io.Discard); err == nil {
		t.Fatal("nil input accepted")
	}
	if _, err := robot.NewStdio("p1", nil); err == nil {
		t.Fatal("nil session accepted")
	}
	s := session(t, io.NopCloser(strings.NewReader("")), io.Discard)
	for _, id := range []game.RobotID{"", "p 1", "p1\np2"} {
		if _, err := robot.NewStdio(id, s); err == nil {
			t.Fatalf("invalid ID %q accepted", id)
		}
	}
}
