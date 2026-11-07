package robot_test

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot"
)

func TestPrintIntegerCommands(t *testing.T) {
	t.Parallel()

	for _, argument := range []string{"0", "-42", "+7", "-9223372036854775808", "9223372036854775807"} {
		t.Run(argument, func(t *testing.T) {
			t.Parallel()

			integer, err := strconv.ParseInt(argument, 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			want := robotmsg.ActionStep{Action: actions.PrintIntegerAction{Integer: integer}}
			input := "p1 print_integer\np1 print_integer extra args\np1 print_integer abc\np1 print_integer 1.5\np1 print_integer 9223372036854775808\np1 print_integer -9223372036854775809\np2 print_integer 3\np1 print_integer " + argument + "\np1 done\n"
			var output bytes.Buffer
			r := stdio(t, "p1", session(t, io.NopCloser(strings.NewReader(input)), &output))

			step, startErr := r.Start()
			done, resumeErr := r.Next(values.UnitValue{})

			if startErr != nil || step.Step != want {
				t.Errorf("print command = %+v, %v, want %+v", step, startErr, want)
			}
			if resumeErr != nil || done.Step != (robotmsg.DoneStep{}) {
				t.Errorf("resume = %+v, %v", done, resumeErr)
			}
			if strings.Count(output.String(), "invalid command for p1") != 7 {
				t.Errorf("invalid commands were not rejected: %s", output.String())
			}
		})
	}
}

func session(t *testing.T, input io.ReadCloser, output io.Writer) *robot.Session {
	t.Helper()
	s, err := robot.NewSession(input, output)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func stdio(t *testing.T, id protocol.RobotID, s *robot.Session) *robot.Stdio {
	t.Helper()
	r, err := robot.NewStdio(id, s)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestInteractiveCommandsAndReturnValues(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	s := session(t, io.NopCloser(strings.NewReader("\nhelp\np2 forward\np1 rotate sideways\np1 forward extra\np1 drill_durability\np1 rotate left\np1 rotate right\np1 back\np1 forward\np1 done\np2 done\n")), &output)
	p1, p2 := stdio(t, "p1", s), stdio(t, "p2", s)
	step, err := p1.Start()
	if err != nil || step.Step != (robotmsg.ActionStep{Action: actions.DrillDurabilityAction{}}) {
		t.Errorf("unexpected initial step: %+v, %v", step, err)
	}
	tests := []struct {
		value values.Value
		want  actions.Action
	}{
		{value: values.IntegerValue{Integer: 3}, want: actions.RotateAction{Rotation: values.Left}},
		{value: values.UnitValue{}, want: actions.RotateAction{Rotation: values.Right}},
		{value: values.UnitValue{}, want: actions.BackAction{}},
		{value: values.BoolValue{Bool: true}, want: actions.ForwardAction{}},
	}
	for _, tt := range tests {
		step, err := p1.Next(tt.value)
		if err != nil || step.Step != (robotmsg.ActionStep{Action: tt.want}) {
			t.Errorf("got %+v, %v; want %+v", step, err, tt.want)
		}
	}
	step, err = p1.Next(values.BoolValue{Bool: false})
	if err != nil || step.Step != (robotmsg.DoneStep{}) {
		t.Errorf("done not accepted: %+v, %v", step, err)
	}
	step, err = p2.Start()
	if err != nil || step.Step != (robotmsg.DoneStep{}) {
		t.Errorf("shared input not routed: %+v, %v", step, err)
	}
	for _, expected := range []string{"invalid command for p1", "p1 => 3", "p1 => ()", "p1 => true", "p1 => false", "p2> "} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("output missing %q: %s", expected, output.String())
		}
	}
}

func TestEOFAppliesToAllRobots(t *testing.T) {
	t.Parallel()
	s := session(t, io.NopCloser(strings.NewReader("p1 forward")), io.Discard)
	p1, p2 := stdio(t, "p1", s), stdio(t, "p2", s)
	step, err := p1.Start()
	if err != nil || step.Step != (robotmsg.ActionStep{Action: actions.ForwardAction{}}) {
		t.Errorf("unterminated last line not read: %+v, %v", step, err)
	}
	step, err = p2.Start()
	if err != nil || step.Step != (robotmsg.DoneStep{}) {
		t.Errorf("EOF not shared with unstarted robot: %+v, %v", step, err)
	}
	step, err = p1.Next(values.BoolValue{Bool: true})
	if err != nil || step.Step != (robotmsg.DoneStep{}) {
		t.Errorf("EOF not shared with running robot: %+v, %v", step, err)
	}
}

func TestDisplaysTerrainReturnValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value values.TerrainValue
		want  string
	}{
		{name: "land", value: values.LandTerrain{Kind: values.PlainLand{}}, want: "terrain(land)"},
		{name: "goal", value: values.LandTerrain{Kind: values.GoalLand{}}, want: "terrain(land, goal)"},
		{name: "checkpoint", value: values.LandTerrain{Kind: values.CheckpointLand{Order: 2}}, want: "terrain(land, checkpoint=2)"},
		{name: "luggage", value: values.LandTerrain{Kind: values.PlainLand{}, HasLuggage: true}, want: "terrain(land, luggage)"},
		{name: "territory with luggage", value: values.LandTerrain{Kind: values.PlainLand{}, HasLuggage: true, Territory: values.OwnTerritory}, want: "terrain(land, luggage, territory=own)"},
		{name: "rock", value: values.RockTerrain{}, want: "terrain(rock)"},
		{name: "sea", value: values.SeaTerrain{}, want: "terrain(sea)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var output bytes.Buffer
			r := stdio(t, "p1", session(t, io.NopCloser(strings.NewReader("p1 forward\np1 done\n")), &output))
			if step, err := r.Start(); err != nil || step.Step != (robotmsg.ActionStep{Action: actions.ForwardAction{}}) {
				t.Errorf("unexpected initial step: %+v, %v", step, err)
			}
			step, err := r.Next(tt.value)
			if err != nil || step.Step != (robotmsg.DoneStep{}) || !strings.Contains(output.String(), "p1 => "+tt.want) {
				t.Errorf("terrain return value not displayed: %+v, %v, %s", step, err, output.String())
			}
		})
	}
}

func TestStartAndResumeLifecycle(t *testing.T) {
	t.Parallel()
	r := stdio(t, "p1", session(t, io.NopCloser(strings.NewReader("p1 forward\np1 done\n")), io.Discard))
	if _, err := r.Next(values.UnitValue{}); err == nil {
		t.Error("resume before start was accepted")
	}
	if step, err := r.Start(); err != nil || step.Step != (robotmsg.ActionStep{Action: actions.ForwardAction{}}) {
		t.Errorf("unexpected initial step: %+v, %v", step, err)
	}
	if _, err := r.Start(); err == nil {
		t.Error("second start was accepted")
	}
	for _, value := range []values.Value{
		nil, (*values.BoolValue)(nil), &values.LandTerrain{Kind: values.PlainLand{}},
		values.LandTerrain{},
		values.LandTerrain{Kind: &values.PlainLand{}},
		values.LandTerrain{Kind: &values.GoalLand{}},
		values.LandTerrain{Kind: &values.CheckpointLand{Order: 2}},
		values.LandTerrain{Kind: (*values.GoalLand)(nil)}, &values.LandTerrain{Kind: values.GoalLand{}},
		values.LandTerrain{Kind: (*values.CheckpointLand)(nil)}, &values.LandTerrain{Kind: values.CheckpointLand{Order: 2}},
		values.LandTerrain{Kind: values.GoalLand{}, Territory: values.OwnTerritory},
		values.LandTerrain{Kind: values.PlainLand{}, Territory: "other"},
	} {
		if _, err := r.Next(value); err == nil {
			t.Errorf("invalid resume value was accepted: %T", value)
		}
	}
	if step, err := r.Next(values.BoolValue{Bool: true}); err != nil || step.Step != (robotmsg.DoneStep{}) {
		t.Errorf("invalid requests consumed input: %+v, %v", step, err)
	}
	if _, err := r.Start(); err == nil {
		t.Error("completed robot was restarted")
	}
	if _, err := r.Next(values.UnitValue{}); err == nil {
		t.Error("completed robot was resumed")
	}
}

type failedIO struct{ err error }

func (f failedIO) Read([]byte) (int, error)  { return 0, f.err }
func (f failedIO) Write([]byte) (int, error) { return 0, f.err }
func (f failedIO) Close() error              { return nil }

func TestIOErrorsAndValidation(t *testing.T) {
	t.Parallel()
	failure := errors.New("I/O failed")
	for _, tc := range []struct {
		input  io.ReadCloser
		output io.Writer
	}{
		{input: failedIO{err: failure}, output: io.Discard},
		{input: io.NopCloser(strings.NewReader("p1 forward\n")), output: failedIO{err: failure}},
	} {
		r := stdio(t, "p1", session(t, tc.input, tc.output))
		if _, err := r.Start(); !errors.Is(err, failure) {
			t.Errorf("I/O error lost: %v", err)
		}
	}
	if _, err := robot.NewSession(nil, io.Discard); err == nil {
		t.Error("nil input accepted")
	}
	if _, err := robot.NewStdio("p1", nil); err == nil {
		t.Error("nil session accepted")
	}
	s := session(t, io.NopCloser(strings.NewReader("")), io.Discard)
	for _, id := range []protocol.RobotID{"", "p 1", "p1\np2"} {
		if _, err := robot.NewStdio(id, s); err == nil {
			t.Errorf("invalid ID %q accepted", id)
		}
	}
}
