package interpreter

import (
	"errors"
	"testing"

	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/internal/testutil"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/builtin"
)

func TestOptionConstructorsAcrossCallsAndActions(t *testing.T) {
	t.Parallel()
	for name, logged := range map[string]bool{"logged": true, "unlogged": false} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			b := &builder{}
			option := bot.OptionType{ElementType: bot.IntegerType{}}
			stack := bot.StackType{ElementType: bot.IntegerType{}}
			program := checked(t, map[bot.Name]fn{
				"identity": b.function([]param{b.param("input", option)}, option, b.ret(b.ref("input"))),
				"main": b.function(nil, bot.UnitType{},
					b.declare("absent", option, b.call("none")),
					b.declare("present", option, b.call("some", b.integer(42))),
					b.declare("nested", bot.OptionType{ElementType: option}, b.call("some", b.call("none"))),
					b.declare("items", stack, b.call("push_back", b.call("new"), b.integer(1))),
					b.declare("saved", bot.OptionType{ElementType: stack}, b.call("some", b.ref("items"))),
					b.assign("items", b.call("push_back", b.ref("items"), b.integer(2))),
					b.call("print_bool", b.call("is_some", b.ref("absent"))),
					b.call("print_bool", b.call("is_some", b.ref("nested"))),
					b.call("print_bool", b.call("is_some", b.call("unwrap", b.ref("nested")))),
					b.call("print_integer", b.call("unwrap", b.call("identity", b.ref("present")))),
					b.call("print_integer", b.call("size", b.call("unwrap", b.ref("saved")))),
					b.assign("present", b.call("none")),
					b.call("print_bool", b.call("is_some", b.ref("present"))), b.ret(b.unit())),
			})
			v := newVMWithLogging(t, program, 0, testLimits(), logged)
			base := v.memory.MemoryUsage()
			var replay testutil.EventReplay
			output, err := v.Start()
			for _, action := range []actions.Action{
				actions.PrintBoolAction{Bool: false}, actions.PrintBoolAction{Bool: true}, actions.PrintBoolAction{Bool: false},
				actions.PrintIntegerAction{Integer: 42}, actions.PrintIntegerAction{Integer: 1}, actions.PrintBoolAction{Bool: false},
			} {
				expectOutput(t, output, err, robotmsg.ActionStep{Action: action})
				for _, e := range output.Events {
					replay.Apply(t, e)
				}
				if !logged && output.Events != nil {
					t.Error("unlogged execution emitted events")
				}
				output, err = v.Next(values.UnitValue{})
			}
			expectOutput(t, output, err, robotmsg.DoneStep{})
			for _, e := range output.Events {
				replay.Apply(t, e)
			}
			replay.AssertEmpty(t)
			assertClean(t, v, base)
		})
	}
}

func TestUnwrapNoneConstructorFailsAndReleasesMemory(t *testing.T) {
	t.Parallel()
	b := &builder{}
	unwrap := b.call("unwrap", b.call("none"))
	main := b.function(nil, bot.UnitType{}, b.call("print_integer", unwrap), b.ret(b.unit()))
	v := newVM(t, checked(t, map[bot.Name]fn{"main": main}), 0)
	base := v.memory.MemoryUsage()
	output, err := v.Start()
	if !errors.Is(err, builtin.ErrUnwrapNone) {
		t.Errorf("Start error = %v, want unwrap-none error", err)
	}
	if output.Step != nil {
		t.Errorf("runtime failure returned a successful step: %#v", output.Step)
	}
	expectExecutionEnd(t, output.Events, robotevent.EndRuntimeError, unwrap.HintID)
	var replay testutil.EventReplay
	for _, e := range output.Events {
		replay.Apply(t, e)
	}
	replay.AssertEmpty(t)
	assertClean(t, v, base)
}
