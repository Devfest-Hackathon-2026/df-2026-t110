package interpreter

import (
	"errors"
	"reflect"
	"testing"

	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/internal/testutil"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/builtin"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
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

func TestMemoryEventsReplayVMReferencesAcrossSuspensionsAndCleanup(t *testing.T) {
	t.Parallel()
	program, _, _, _ := factorial(t)
	for _, name := range []string{"completed", "interrupted", "step limit", "memory limit", "depth limit", "runtime error"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			limits := testLimits()
			selected := program
			switch name {
			case "step limit":
				limits.MaxSteps = 12
			case "memory limit":
				limits.MaxMemoryBytes = 1
			case "depth limit":
				limits.MaxCallDepth = 4
			case "runtime error":
				b := &builder{}
				selected = checked(t, map[bot.Name]fn{"main": b.function(nil, bot.UnitType{}, b.call("div", b.integer(1), b.integer(0)), b.ret(b.unit()))})
			}
			v := newLimitedVM(t, selected, 0, limits)
			var replay testutil.EventReplay
			ids := make(map[memory.ID]struct{})
			check := func(events []robotevent.PicolangEvent) {
				for _, event := range events {
					replay.Apply(t, event)
					if allocation, ok := event.(robotevent.MemoryAllocateEvent); ok {
						ids[allocation.ID] = struct{}{}
					}
				}
				if replay.Memory.MemoryUsage() != v.memory.MemoryUsage() {
					t.Error("replayed memory usage differs from VM")
				}
				for id := range ids {
					actual, exists := v.memory.Get(id)
					decoded, replayExists := replay.Memory.Get(id)
					if exists != replayExists || !reflect.DeepEqual(actual, decoded) {
						t.Errorf("ID %d: VM value %#v, replay value %#v", id, actual, decoded)
					}
					refs, _ := v.memory.RefCount(id)
					decodedRefs, _ := replay.Memory.RefCount(id)
					if refs != decodedRefs {
						t.Errorf("ID %d: VM references %d, replay references %d", id, refs, decodedRefs)
					}
				}
			}
			output, err := v.Start()
			check(output.Events)
			if name == "interrupted" {
				if err != nil {
					t.Fatal(err)
				}
				if err := v.Close(); err != nil {
					t.Fatal(err)
				}
				check(v.Collect())
			} else if name == "completed" {
				for range 3 {
					if err != nil {
						t.Fatal(err)
					}
					if _, done := output.Step.(robotmsg.DoneStep); done {
						break
					}
					output, err = v.Next(values.UnitValue{})
					check(output.Events)
				}
				if _, done := output.Step.(robotmsg.DoneStep); !done {
					t.Error("VM did not complete")
				}
			} else if err == nil {
				t.Error("expected execution failure")
			}
			replay.AssertEmpty(t)
		})
	}
}
