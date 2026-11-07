// Package robot provides an interactive text-command implementation of values.Robot.
package robot

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

// CommandUsage lists the commands accepted by a stdio robot session.
const CommandUsage = `
<id> forward | back | rotate left | rotate right |
 drill | drill_durability | 
 search_now | search_front | 
 print_integer <integer> | print_bool true/false | 
 done
help`

// Session owns input and shares it between robots. Calls to Robot.Start and
// Robot.Next must be sequential. Close releases the input resource.
type Session struct {
	input     io.ReadCloser
	output    io.Writer
	scanner   *bufio.Scanner
	closeOnce sync.Once
	closeErr  error
	eof       bool
}

func NewSession(input io.ReadCloser, output io.Writer) (*Session, error) {
	if input == nil || output == nil {
		return nil, errors.New("input and output are required")
	}
	return &Session{input: input, output: output, scanner: bufio.NewScanner(input)}, nil
}

func (s *Session) Close() error {
	s.closeOnce.Do(func() { s.closeErr = s.input.Close() })
	return s.closeErr
}

func (s *Session) readLine() (string, error) {
	if s.scanner.Scan() {
		return s.scanner.Text(), nil
	}
	if err := s.scanner.Err(); err != nil {
		return "", err
	}
	return "", io.EOF
}

type Stdio struct {
	id      protocol.RobotID
	session *Session
	phase   stdioPhase
}

type stdioPhase interface{ stdioPhase() }

type unstartedStdio struct{}
type runningStdio struct{}
type finishedStdio struct{}

func (unstartedStdio) stdioPhase() {}
func (runningStdio) stdioPhase()   {}
func (finishedStdio) stdioPhase()  {}

func NewStdio(id protocol.RobotID, session *Session) (*Stdio, error) {
	if id == "" || strings.IndexFunc(string(id), unicode.IsSpace) >= 0 || session == nil {
		return nil, errors.New("a nonempty whitespace-free robot ID and session are required")
	}
	return &Stdio{id: id, session: session, phase: unstartedStdio{}}, nil
}

func (r *Stdio) Start() (robotmsg.RobotOutput, error) {
	if _, fresh := r.phase.(unstartedStdio); !fresh {
		return robotmsg.RobotOutput{}, errors.New("robot has already started")
	}
	r.phase = runningStdio{}
	step, err := r.readStep()
	return robotmsg.RobotOutput{Step: step}, err
}

func (r *Stdio) Next(value values.Value) (robotmsg.RobotOutput, error) {
	if _, running := r.phase.(runningStdio); !running {
		return robotmsg.RobotOutput{}, errors.New("robot must be running to resume")
	}
	if !values.ValidValue(value) {
		return robotmsg.RobotOutput{}, errors.New("resume requires a return value")
	}
	if _, err := fmt.Fprintf(r.session.output, "%s => %s\n", r.id, formatValue(value)); err != nil {
		return robotmsg.RobotOutput{}, err
	}
	step, err := r.readStep()
	return robotmsg.RobotOutput{Step: step}, err
}

func (r *Stdio) readStep() (robotmsg.Step, error) {
	for !r.session.eof {
		if _, err := fmt.Fprintf(r.session.output, "%s> ", r.id); err != nil {
			return nil, err
		}
		text, err := r.session.readLine()
		if errors.Is(err, io.EOF) {
			r.session.eof = true
			break
		}
		if err != nil {
			return nil, err
		}
		fields := strings.Fields(text)
		if len(fields) == 1 && fields[0] == "help" {
			if _, err := fmt.Fprintln(r.session.output, CommandUsage); err != nil {
				return nil, err
			}
			continue
		}
		step, ok := parse(r.id, fields)
		if ok {
			if _, done := step.(robotmsg.DoneStep); done {
				r.phase = finishedStdio{}
			}
			return step, nil
		}
		if _, err := fmt.Fprintf(r.session.output, "invalid command for %s; %s\n", r.id, CommandUsage); err != nil {
			return nil, err
		}
	}
	r.phase = finishedStdio{}
	return robotmsg.DoneStep{}, nil
}

func parse(id protocol.RobotID, fields []string) (robotmsg.Step, bool) {
	if len(fields) < 2 || fields[0] != string(id) {
		return nil, false
	}
	if len(fields) == 2 {
		switch fields[1] {
		case "done":
			return robotmsg.DoneStep{}, true
		case "forward":
			return robotmsg.ActionStep{Action: actions.ForwardAction{}}, true
		case "back":
			return robotmsg.ActionStep{Action: actions.BackAction{}}, true
		case "drill_durability":
			return robotmsg.ActionStep{Action: actions.DrillDurabilityAction{}}, true
		case "drill":
			return robotmsg.ActionStep{Action: actions.DrillAction{}}, true
		case "search_now":
			return robotmsg.ActionStep{Action: actions.SearchNowAction{}}, true
		case "search_front":
			return robotmsg.ActionStep{Action: actions.SearchFrontAction{}}, true
		}
	}
	if len(fields) == 3 && fields[1] == "rotate" && (fields[2] == string(values.Left) || fields[2] == string(values.Right)) {
		return robotmsg.ActionStep{Action: actions.RotateAction{Rotation: values.Rotation(fields[2])}}, true
	}
	if len(fields) == 3 && fields[1] == "print_integer" {
		integer, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil {
			return nil, false
		}
		return robotmsg.ActionStep{Action: actions.PrintIntegerAction{Integer: integer}}, true
	}
	if len(fields) == 3 && fields[1] == "print_bool" && (fields[2] == "true" || fields[2] == "false") {
		return robotmsg.ActionStep{Action: actions.PrintBoolAction{Bool: fields[2] == "true"}}, true
	}
	return nil, false
}

func formatValue(value values.Value) string {
	switch value := value.(type) {
	case values.BoolValue:
		return strconv.FormatBool(value.Bool)
	case values.IntegerValue:
		return strconv.FormatInt(value.Integer, 10)
	case values.UnitValue:
		return "()"
	case values.LandTerrain:
		attributes := []string{"land"}
		switch kind := value.Kind.(type) {
		case values.GoalLand:
			attributes = append(attributes, "goal")
		case values.CheckpointLand:
			attributes = append(attributes, fmt.Sprintf("checkpoint=%d", kind.Order))
		}
		if value.HasLuggage {
			attributes = append(attributes, "luggage")
		}
		if value.Territory != values.NoTerritory {
			attributes = append(attributes, "territory="+string(value.Territory))
		}
		return "terrain(" + strings.Join(attributes, ", ") + ")"
	case values.RockTerrain:
		return "terrain(rock)"
	case values.SeaTerrain:
		return "terrain(sea)"
	}
	return "<invalid value>"
}
