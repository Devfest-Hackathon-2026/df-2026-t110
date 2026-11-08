package parse_test

import (
	"bytes"
	json "encoding/json/v2"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	robotmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/index"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/parse"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/typecheck"
)

const minimal = `{"picolangVersion":0,"picolang":{"v0":{"program":{"main":{"kind":"function","hintId":"00000000-0000-4000-8000-000000000001","params":[],"returnType":"unit","body":{"kind":"block","hintId":"00000000-0000-4000-8000-000000000002","stmts":[{"kind":"return","hintId":"00000000-0000-4000-8000-000000000003","value":{"kind":"literal","hintId":"00000000-0000-4000-8000-000000000004","type":"unit","value":null}}]}}}}}}`

type fixture struct{ next int }

func (f *fixture) node(kind string, fields ...any) map[string]any {
	f.next++
	value := map[string]any{"kind": kind, "hintId": fmt.Sprintf("00000000-0000-4000-8000-%012x", f.next)}
	for i := 0; i+1 < len(fields); i += 2 {
		value[fields[i].(string)] = fields[i+1]
	}
	return value
}

func (f *fixture) literal(kind string, value any) map[string]any {
	return f.node("literal", "type", kind, "value", value)
}

func (f *fixture) ident(name string) map[string]any { return f.node("ident", "name", name) }

func (f *fixture) call(name string, args ...any) map[string]any {
	if args == nil {
		args = []any{}
	}
	return f.node("call", "func", name, "args", args)
}

func (f *fixture) block(stmts ...any) map[string]any {
	if stmts == nil {
		stmts = []any{}
	}
	return f.node("block", "stmts", stmts)
}

func source(t *testing.T, functions map[string]any) []byte {
	t.Helper()
	data, err := json.Marshal(map[string]any{"picolangVersion": 0, "picolang": map[string]any{"v0": map[string]any{"program": functions}}})
	if err != nil {
		t.Fatalf("prepare program JSON: %v", err)
	}
	return data
}

func hint(node map[string]any) bot.Source {
	return bot.Source{HintID: bot.HintID(node["hintId"].(string))}
}

func TestMinimalProgram(t *testing.T) {
	t.Parallel()
	input := []byte(minimal)
	before := bytes.Clone(input)
	program, err := parse.Parse(input)
	if err != nil {
		t.Errorf("Parse: %v", err)
		return
	}
	want := bot.ParsedProgram{Functions: map[bot.Name]bot.Function[bot.Name, bot.Untyped]{"main": {
		Source: bot.Source{HintID: "00000000-0000-4000-8000-000000000001"},
		Params: []bot.Param[bot.Name]{}, ReturnType: bot.UnitType{},
		Body: bot.Block[bot.Name, bot.Untyped]{Source: bot.Source{HintID: "00000000-0000-4000-8000-000000000002"}, Stmts: []bot.Stmt[bot.Name, bot.Untyped]{
			bot.Return[bot.Name, bot.Untyped]{Source: bot.Source{HintID: "00000000-0000-4000-8000-000000000003"}, Value: bot.LiteralExpr[bot.Name, bot.Untyped]{Source: bot.Source{HintID: "00000000-0000-4000-8000-000000000004"}, Value: bot.UnitLiteral{}}},
		}},
	}}}
	if !reflect.DeepEqual(program, want) {
		t.Errorf("parsed AST = %#v, want %#v", program, want)
	}
	if !bytes.Equal(input, before) {
		t.Error("Parse modified its input")
	}
}

func TestAllSyntaxVariants(t *testing.T) {
	t.Parallel()
	f := &fixture{}
	stack := map[string]any{"kind": "stack", "elementType": "integer"}
	nested := map[string]any{"kind": "stack", "elementType": stack}
	declaration := f.node("declare", "type", nested, "ident", f.ident("matrix"), "value", f.call("new"))
	assignment := f.node("assign", "ident", f.ident("matrix"), "value", f.call("new"))
	conditional := f.node("if", "condition", f.literal("bool", true), "then", f.block(f.node("break")), "else", f.block(f.node("continue")))
	withoutElse := f.node("if", "condition", f.literal("bool", false), "then", f.block())
	loop := f.node("while", "condition", f.literal("bool", false), "body", f.block())
	counted := f.node("for", "ident", f.ident("i"), "start", f.literal("integer", 1), "end", f.literal("integer", 3), "body", f.block())
	call := f.call("rotate", f.literal("rotation", "left"))
	reference := f.ident("matrix")
	ret := f.node("return", "value", f.literal("unit", nil))
	stmts := []any{declaration, assignment, conditional, withoutElse, loop, counted, call, reference}
	for _, target := range []string{"robot", "ally", "enemy", "rock", "luggage", "sea", "goal", "territory"} {
		stmts = append(stmts, f.literal("distance_target", target))
	}
	stmts = append(stmts, f.literal("rotation", "right"), ret)
	params := []any{}
	wantTypes := []bot.Type{bot.BoolType{}, bot.IntegerType{}, bot.UnitType{}, bot.TerrainType{}, bot.RotationType{}, bot.DistanceTargetType{}, bot.StackType{ElementType: bot.StackType{ElementType: bot.IntegerType{}}}}
	for i, typ := range []any{"bool", "integer", "unit", "terrain", "rotation", "distance_target", nested} {
		params = append(params, f.node("param", "ident", f.ident(fmt.Sprintf("p%d", i)), "type", typ))
	}
	body := f.block(stmts...)
	fn := f.node("function", "params", params, "returnType", "unit", "body", body)
	// Missing main, break outside a loop and names are deliberately left for the
	// semantic stages; they are valid JSON syntax and must survive parsing.
	program, err := parse.Parse(source(t, map[string]any{"検査": fn}))
	if err != nil {
		t.Errorf("Parse: %v", err)
		return
	}
	got := program.Functions["検査"]
	if got.Source != hint(fn) || got.Body.Source != hint(body) || len(got.Params) != len(params) || len(got.Body.Stmts) != len(stmts) {
		t.Errorf("function structure was not preserved: %+v", got)
		return
	}
	for i, param := range got.Params {
		raw := params[i].(map[string]any)
		if param.Source != hint(raw) || param.Ident.Source != hint(raw["ident"].(map[string]any)) || param.Ident.Variable != bot.Name(fmt.Sprintf("p%d", i)) || !reflect.DeepEqual(param.Type, wantTypes[i]) {
			t.Errorf("parameter %d = %+v", i, param)
		}
	}
	wantKinds := []any{bot.Declaration[bot.Name, bot.Untyped]{}, bot.Assignment[bot.Name, bot.Untyped]{}, bot.If[bot.Name, bot.Untyped]{}, bot.If[bot.Name, bot.Untyped]{}, bot.While[bot.Name, bot.Untyped]{}, bot.For[bot.Name, bot.Untyped]{}, bot.CallExpr[bot.Name, bot.Untyped]{}, bot.IdentExpr[bot.Name, bot.Untyped]{}}
	for i, want := range wantKinds {
		if reflect.TypeOf(got.Body.Stmts[i]) != reflect.TypeOf(want) || bot.SourceOf(got.Body.Stmts[i], bot.Source{}) != hint(stmts[i].(map[string]any)) {
			t.Errorf("statement %d = %#v", i, got.Body.Stmts[i])
		}
	}
	if node, ok := got.Body.Stmts[0].(bot.Declaration[bot.Name, bot.Untyped]); !ok || !reflect.DeepEqual(node.Type, wantTypes[6]) || node.Ident.Source != hint(declaration["ident"].(map[string]any)) {
		t.Errorf("nested stack declaration = %#v", got.Body.Stmts[0])
	}
	if node, ok := got.Body.Stmts[2].(bot.If[bot.Name, bot.Untyped]); !ok || node.Else == nil || len(node.Then.Stmts) != 1 || len(node.Else.Stmts) != 1 {
		t.Errorf("if/else = %#v", got.Body.Stmts[2])
	}
	if node, ok := got.Body.Stmts[3].(bot.If[bot.Name, bot.Untyped]); !ok || node.Else != nil {
		t.Errorf("omitted else = %#v", got.Body.Stmts[3])
	}
	for i, target := range []values.DistanceTarget{values.DistanceRobot, values.DistanceAlly, values.DistanceEnemy, values.DistanceRock, values.DistanceLuggage, values.DistanceSea, values.DistanceGoal, values.DistanceTerritory} {
		node, ok := got.Body.Stmts[8+i].(bot.LiteralExpr[bot.Name, bot.Untyped])
		if !ok || node.Value != (bot.DistanceTargetLiteral{Value: target}) {
			t.Errorf("distance target %d = %#v", i, node)
		}
	}
}

func TestInvalidJSONAndSyntax(t *testing.T) {
	t.Parallel()
	replace := func(old, new string) string { return strings.Replace(minimal, old, new, 1) }
	for name, input := range map[string]string{
		"empty": "", "truncated": minimal[:len(minimal)-1], "null": "null", "array": "[]",
		"trailing JSON": minimal + `{}`, "invalid UTF8": replace(`"main"`, "\"\xff\""),
		"duplicate version":          replace(`"picolangVersion":0`, `"picolangVersion":0,"picolangVersion":0`),
		"duplicate escaped function": replace(`"main":`, `"main":{},"\u006dain":`),
		"duplicate node field":       replace(`"kind":"function"`, `"kind":"function","kind":"function"`),
		"version missing":            replace(`"picolangVersion":0,`, ""),
		"unsupported version":        replace(`"picolangVersion":0`, `"picolangVersion":1`),
		"version string":             replace(`"picolangVersion":0`, `"picolangVersion":"0"`),
		"wrong version object":       replace(`"v0":`, `"v1":`),
		"unknown root field":         replace(`"picolangVersion":0`, `"extra":1,"picolangVersion":0`),
		"unknown function field":     replace(`"params":[]`, `"params":[],"extra":true`),
		"missing params":             replace(`"params":[],`, ""),
		"null params":                replace(`"params":[]`, `"params":null`),
		"null body":                  replace(`"body":{`, `"body":null,"extra":{`),
		"null statements":            replace(`"stmts":[`, `"stmts":null,"extra":[`),
		"null statement":             replace(`"stmts":[{`, `"stmts":[null,{`),
		"missing unit value":         replace(`"type":"unit","value":null`, `"type":"unit"`),
		"non-null unit":              replace(`"value":null`, `"value":false`),
		"unknown literal":            replace(`"type":"unit","value":null`, `"type":"terrain","value":null`),
		"integer string":             replace(`"type":"unit","value":null`, `"type":"integer","value":"3"`),
		"invalid rotation":           replace(`"type":"unit","value":null`, `"type":"rotation","value":"up"`),
		"invalid distance target":    replace(`"type":"unit","value":null`, `"type":"distance_target","value":"robots"`),
		"unknown return type":        replace(`"returnType":"unit"`, `"returnType":"float"`),
		"missing stack element":      replace(`"returnType":"unit"`, `"returnType":{"kind":"stack"}`),
		"old vector type":            replace(`"returnType":"unit"`, `"returnType":{"kind":"vector","elementType":"integer"}`),
		"unknown statement":          replace(`"kind":"return"`, `"kind":"unknown"`),
		"expression as block":        replace(`"kind":"block"`, `"kind":"call"`),
		"invalid hint":               replace(`00000000-0000-4000-8000-000000000001`, `not-a-uuid`),
		"missing hint":               replace(`"hintId":"00000000-0000-4000-8000-000000000001",`, ""),
		"duplicate hints":            replace(`00000000-0000-4000-8000-000000000004`, `00000000-0000-4000-8000-000000000001`),
		"empty function name":        replace(`"main":`, `"":`),
		"null call arguments":        strings.Replace(replace(`"kind":"literal"`, `"kind":"call"`), `"type":"unit","value":null`, `"args":null,"func":"new"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			program, err := parse.Parse([]byte(input))
			if err == nil || program.Functions != nil {
				t.Errorf("invalid input returned %#v, %v", program, err)
			}
		})
	}
}

func TestHintUniquenessIncludesAllNodesAndFunctions(t *testing.T) {
	t.Parallel()
	for _, location := range []string{"functions", "parameter and ident", "UUID letter case"} {
		t.Run(location, func(t *testing.T) {
			t.Parallel()
			f := &fixture{}
			param := f.node("param", "ident", f.ident("x"), "type", "integer")
			body := f.block(f.node("return", "value", f.literal("unit", nil)))
			fn := f.node("function", "params", []any{param}, "returnType", "unit", "body", body)
			functions := map[string]any{"a": fn}
			switch location {
			case "functions":
				functions["b"] = fn
			case "parameter and ident":
				param["hintId"] = param["ident"].(map[string]any)["hintId"]
			case "UUID letter case":
				fn["hintId"], body["hintId"] = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA"
			}
			program, err := parse.Parse(source(t, functions))
			if err == nil || !strings.Contains(err.Error(), "duplicate hintId") || program.Functions != nil {
				t.Errorf("duplicate hints returned %#v, %v", program, err)
			}
		})
	}
}

func TestEmptyProgramIsLeftForSemanticAnalysis(t *testing.T) {
	t.Parallel()
	program, err := parse.Parse(source(t, map[string]any{}))
	if err != nil || program.Functions == nil || len(program.Functions) != 0 {
		t.Errorf("empty program = %#v, %v", program, err)
	}
}

func TestIntegerPrecisionAndNotation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		text string
		want int64
	}{
		{"9223372036854775807", math.MaxInt64}, {"-9223372036854775808", math.MinInt64},
		{"9007199254740993", 9007199254740993}, {"9007199254740993.0", 9007199254740993},
		{"9.007199254740993e15", 9007199254740993}, {"100e-2", 1}, {"-0.0e999999999999999999999999", 0},
	} {
		t.Run(tc.text, func(t *testing.T) {
			t.Parallel()
			f := &fixture{}
			literal := f.literal("integer", "NUMBER")
			fn := f.node("function", "params", []any{}, "returnType", "integer", "body", f.block(f.node("return", "value", literal)))
			input := bytes.Replace(source(t, map[string]any{"main": fn}), []byte(`"NUMBER"`), []byte(tc.text), 1)
			program, err := parse.Parse(input)
			if err != nil {
				t.Errorf("Parse: %v", err)
				return
			}
			ret, ok := program.Functions["main"].Body.Stmts[0].(bot.Return[bot.Name, bot.Untyped])
			if !ok {
				t.Errorf("return = %#v", program.Functions["main"].Body.Stmts[0])
				return
			}
			value, ok := ret.Value.(bot.LiteralExpr[bot.Name, bot.Untyped])
			if !ok || value.Value != (bot.IntegerLiteral{Value: tc.want}) {
				t.Errorf("literal = %#v, want %d", ret.Value, tc.want)
			}
		})
	}
	for _, text := range []string{"9223372036854775808", "-9223372036854775809", "1.5", "1.00000000000000000000000000000000000000001", "1e1000000000", "1e-1000000000"} {
		t.Run("reject/"+text, func(t *testing.T) {
			t.Parallel()
			input := strings.Replace(minimal, `"type":"unit","value":null`, `"type":"integer","value":`+text, 1)
			if program, err := parse.Parse([]byte(input)); err == nil || program.Functions != nil {
				t.Errorf("accepted %s: %#v, %v", text, program, err)
			}
		})
	}
}

func TestJSONToVM(t *testing.T) {
	t.Parallel()
	f := &fixture{}
	declare := f.node("declare", "type", "integer", "ident", f.ident("count"), "value", f.literal("integer", 0))
	loop := f.node("while", "condition", f.call("lt", f.ident("count"), f.literal("integer", 3)), "body", f.block(
		f.node("assign", "ident", f.ident("count"), "value", f.call("add", f.ident("count"), f.literal("integer", 1))),
	))
	fn := f.node("function", "params", []any{}, "returnType", "unit", "body", f.block(declare, loop, f.call("forward"), f.node("return", "value", f.literal("unit", nil))))
	parsed, err := parse.Parse(source(t, map[string]any{"main": fn}))
	if err != nil {
		t.Errorf("Parse: %v", err)
		return
	}
	indexed, err := index.Index(parsed)
	if err != nil {
		t.Errorf("Index: %v", err)
		return
	}
	typed, err := typecheck.TypeCheck(indexed)
	if err != nil {
		t.Errorf("TypeCheck: %v", err)
		return
	}
	vm, err := interpreter.New(typed, 42, interpreter.Limits{MaxSteps: 100, MaxMemoryBytes: 64 << 10, MaxCallDepth: 10}, true)
	if err != nil {
		t.Errorf("New VM: %v", err)
		return
	}
	t.Cleanup(func() {
		if err := vm.Close(); err != nil {
			t.Errorf("Close VM: %v", err)
		}
	})
	output, err := vm.Start()
	if err != nil || !reflect.DeepEqual(output.Step, robotmsg.ActionStep{Action: actions.ForwardAction{}}) {
		t.Errorf("Start = %#v, %v", output.Step, err)
		return
	}
	output, err = vm.Next(values.BoolValue{Bool: true})
	if err != nil || output.Step != (robotmsg.DoneStep{}) {
		t.Errorf("Next = %#v, %v", output.Step, err)
	}
}

func FuzzParse(f *testing.F) {
	for _, seed := range []string{minimal, "null", `{}`, strings.Replace(minimal, `"value":null`, `"value":false`, 1)} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		program, err := parse.Parse(input)
		if err != nil && program.Functions != nil {
			t.Errorf("failed Parse returned a partial AST: %#v, %v", program, err)
		}
	})
}
