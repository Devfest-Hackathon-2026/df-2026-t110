package interpreter

import (
	"errors"
	"testing"

	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/builtin"
)

func TestUnloggedFailuresPreserveErrorsAndCleanup(t *testing.T) {
	t.Parallel()
	b := &builder{}
	program := checked(t, map[bot.Name]fn{
		"main":   b.function(nil, bot.UnitType{}, b.declare("n", bot.IntegerType{}, b.integer(1)), b.call("divide", b.ref("n")), b.ret(b.unit())),
		"divide": b.function([]param{b.param("n", bot.IntegerType{})}, bot.IntegerType{}, b.ret(b.call("div", b.ref("n"), b.integer(0)))),
	})
	steps, memory, depth := testLimits(), testLimits(), testLimits()
	steps.MaxSteps, memory.MaxMemoryBytes, depth.MaxCallDepth = 1, 1, 1
	for _, tc := range []struct {
		name   string
		limits Limits
		want   error
	}{
		{"runtime", testLimits(), builtin.ErrDivisionByZero},
		{"steps", steps, ErrStepLimit},
		{"memory", memory, ErrMemoryLimit},
		{"depth", depth, ErrCallDepthLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := newVMWithLogging(t, program, 42, tc.limits, false)
			base := v.memory.MemoryUsage()
			output, err := v.Start()
			if !errors.Is(err, tc.want) {
				t.Errorf("Start error = %v, want %v", err, tc.want)
			}
			if output.Step != nil || output.Events != nil {
				t.Errorf("unlogged failure returned output: %#v", output)
			}
			if err := v.Close(); err != nil {
				t.Errorf("Close: %v", err)
			}
			if events := v.Collect(); events != nil {
				t.Errorf("unlogged cleanup returned events: %#v", events)
			}
			assertClean(t, v, base)
		})
	}
}

func TestUnloggedCloseReleasesSuspendedFrames(t *testing.T) {
	t.Parallel()
	program, _, _, _ := factorial(t)
	v := newVMWithLogging(t, program, 42, testLimits(), false)
	base := v.memory.MemoryUsage()
	output, err := v.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if v.phase != awaiting || output.Events != nil {
		t.Errorf("unlogged suspension = phase %v, output %#v", v.phase, output)
	}
	if err := v.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	if events := v.Collect(); events != nil {
		t.Errorf("unlogged interruption returned events: %#v", events)
	}
	assertClean(t, v, base)
}
