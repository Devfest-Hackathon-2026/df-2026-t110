package parse_test

import (
	"reflect"
	"strings"
	"testing"

	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/index"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/parse"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/typecheck"
)

func TestParseAndCheckNestedOptions(t *testing.T) {
	t.Parallel()
	f := &fixture{}
	option := map[string]any{"kind": "option", "elementType": "integer"}
	stack := map[string]any{"kind": "stack", "elementType": option}
	nested := map[string]any{"kind": "option", "elementType": stack}
	want := bot.OptionType{ElementType: bot.StackType{ElementType: bot.OptionType{ElementType: bot.IntegerType{}}}}
	functions := map[string]any{
		"main": f.node("function", "params", []any{}, "returnType", "unit", "body", f.block(
			f.node("declare", "type", stack, "ident", f.ident("options"), "value", f.call("new")),
			f.node("assign", "ident", f.ident("options"), "value", f.call("push_back", f.ident("options"), f.call("none"))),
			f.node("assign", "ident", f.ident("options"), "value", f.call("push_back", f.ident("options"), f.call("some", f.literal("integer", 42)))),
			f.call("identity", f.call("some", f.ident("options"))),
			f.node("return", "value", f.literal("unit", nil)))),
		"identity": f.node("function", "params", []any{f.node("param", "ident", f.ident("input"), "type", nested)}, "returnType", nested, "body", f.block(
			f.call("is_some", f.ident("input")),
			f.call("unwrap", f.ident("input")),
			f.node("return", "value", f.ident("input")))),
	}
	p, err := parse.Parse(source(t, functions))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := p.Functions["identity"]; !reflect.DeepEqual(got.ReturnType, want) || !reflect.DeepEqual(got.Params[0].Type, want) {
		t.Errorf("nested option types = %#v / %#v, want %#v", got.ReturnType, got.Params[0].Type, want)
	}
	indexed, err := index.Index(p)
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
	if _, err := typecheck.TypeCheck(indexed); err != nil {
		t.Errorf("TypeCheck: %v", err)
	}
}

func TestInvalidOptionTypes(t *testing.T) {
	t.Parallel()
	for _, typ := range []string{
		`"option"`, `{"kind":"option"}`, `{"kind":"option","elementType":null}`,
		`{"kind":"option","elementType":"unknown"}`, `{"kind":"option","elementType":"integer","extra":true}`,
		`{"kind":"option","elementType":{"kind":"option"}}`,
	} {
		t.Run(typ, func(t *testing.T) {
			t.Parallel()
			input := strings.Replace(minimal, `"returnType":"unit"`, `"returnType":`+typ, 1)
			if _, err := parse.Parse([]byte(input)); err == nil {
				t.Errorf("accepted invalid option type %s", typ)
			}
		})
	}
}
