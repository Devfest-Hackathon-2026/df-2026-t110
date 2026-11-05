// Package robot provides an interactive text-command implementation of game.Robot.
package robot

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/dvpico/Programming-Learning-AR/Server/game"
)

const help = "<id> forward | back | rotate left | rotate right | drill_durability | done; help\n"

type line struct {
	text string
	err  error
}

// Session owns input and shares it between robots. Calls to Robot.Next must be
// sequential. Closing input must unblock Read (as os.File and io.PipeReader do).
// One reader goroutine is created lazily; Close or cancellation releases it.
type Session struct {
	input     io.ReadCloser
	output    io.Writer
	lines     chan line
	closed    chan struct{}
	readOnce  sync.Once
	closeOnce sync.Once
	closeErr  error
	eof       bool
}

func NewSession(input io.ReadCloser, output io.Writer) (*Session, error) {
	if input == nil || output == nil {
		return nil, errors.New("input and output are required")
	}
	return &Session{input: input, output: output, lines: make(chan line), closed: make(chan struct{})}, nil
}

func (s *Session) Close() error {
	s.closeOnce.Do(func() {
		close(s.closed)
		s.closeErr = s.input.Close()
	})
	return s.closeErr
}

func (s *Session) read() {
	defer close(s.lines)
	scanner := bufio.NewScanner(s.input)
	for scanner.Scan() {
		select {
		case s.lines <- line{text: scanner.Text()}:
		case <-s.closed:
			return
		}
	}
	if err := scanner.Err(); err != nil {
		select {
		case s.lines <- line{err: err}:
		case <-s.closed:
		}
	}
}

func (s *Session) readLine(ctx context.Context) (string, error) {
	s.readOnce.Do(func() { go s.read() })
	select {
	case <-ctx.Done():
		_ = s.Close()
		return "", ctx.Err()
	case <-s.closed:
		return "", io.ErrClosedPipe
	case value, ok := <-s.lines:
		if !ok {
			return "", io.EOF
		}
		return value.text, value.err
	}
}

type Stdio struct {
	id      game.RobotID
	session *Session
	done    bool
}

func NewStdio(id game.RobotID, session *Session) (*Stdio, error) {
	if id == "" || strings.IndexFunc(string(id), unicode.IsSpace) >= 0 || session == nil {
		return nil, errors.New("a nonempty whitespace-free robot ID and session are required")
	}
	return &Stdio{id: id, session: session}, nil
}

func (r *Stdio) Next(ctx context.Context, value *game.Value) (game.Step, error) {
	if err := ctx.Err(); err != nil {
		return game.Step{}, err
	}
	if r.done {
		return game.Step{Done: true}, nil
	}
	if value != nil {
		if _, err := fmt.Fprintf(r.session.output, "%s => %s\n", r.id, formatValue(*value)); err != nil {
			return game.Step{}, err
		}
	}
	for !r.session.eof {
		if _, err := fmt.Fprintf(r.session.output, "%s> ", r.id); err != nil {
			return game.Step{}, err
		}
		text, err := r.session.readLine(ctx)
		if errors.Is(err, io.EOF) {
			r.session.eof = true
			break
		}
		if err != nil {
			return game.Step{}, err
		}
		if err := ctx.Err(); err != nil {
			return game.Step{}, err
		}
		fields := strings.Fields(text)
		if len(fields) == 1 && fields[0] == "help" {
			if _, err := io.WriteString(r.session.output, help); err != nil {
				return game.Step{}, err
			}
			continue
		}
		step, ok := parse(r.id, fields)
		if ok {
			r.done = step.Done
			return step, nil
		}
		if _, err := fmt.Fprintf(r.session.output, "invalid command for %s; %s", r.id, help); err != nil {
			return game.Step{}, err
		}
	}
	r.done = true
	return game.Step{Done: true}, nil
}

func parse(id game.RobotID, fields []string) (game.Step, bool) {
	if len(fields) < 2 || fields[0] != string(id) {
		return game.Step{}, false
	}
	if len(fields) == 2 {
		switch fields[1] {
		case "done":
			return game.Step{Done: true}, true
		case "forward":
			return game.Step{Action: game.ForwardAction{}}, true
		case "back":
			return game.Step{Action: game.BackAction{}}, true
		case "drill_durability":
			return game.Step{Action: game.DrillDurabilityAction{}}, true
		}
	}
	if len(fields) == 3 && fields[1] == "rotate" && (fields[2] == string(game.Left) || fields[2] == string(game.Right)) {
		return game.Step{Action: game.RotateAction{Rotation: game.Rotation(fields[2])}}, true
	}
	return game.Step{}, false
}

func formatValue(value game.Value) string {
	switch value.Kind {
	case game.Bool:
		return strconv.FormatBool(value.Bool)
	case game.Integer:
		return strconv.FormatInt(value.Integer, 10)
	case game.Unit:
		return "()"
	case game.Field:
		return fmt.Sprintf("field(%+v)", value.Field)
	default:
		return "<invalid value>"
	}
}
