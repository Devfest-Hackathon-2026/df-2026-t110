package router

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"math"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/parse"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/google/uuid"
)

func TestGeneratedHintIDsUseSchemaModels(t *testing.T) {
	t.Parallel()
	const placeholder schema.HintId = "00000000-0000-4000-8000-000000000000"
	ident := schema.Ident{Kind: schema.IdentKind, HintId: placeholder, Name: "n"}
	integer := schema.IntegerLiteral{Kind: "literal", HintId: placeholder, Type: "integer", Value: math.MaxInt64}
	boolean := schema.BoolLiteral{Kind: "literal", HintId: placeholder, Type: "bool", Value: true}
	block := func(stmts ...schema.Stmt) schema.Block {
		return schema.Block{Kind: schema.BlockKind, HintId: placeholder, Stmts: stmts}
	}
	other := block(schema.Continue{Kind: schema.ContinueKind, HintId: placeholder})
	fn := schema.PicolangFunctionDefinition{
		Kind: schema.PicolangFunctionDefinitionKind, HintId: placeholder,
		Params: []schema.Param{{Kind: schema.ParamKind, HintId: placeholder, Ident: ident,
			Type: schema.OptionType{Kind: "option", ElementType: schema.StackType{Kind: "stack", ElementType: "integer"}}}},
		ReturnType: "unit",
		Body: block(
			schema.Declaration{Kind: schema.DeclarationKind, HintId: placeholder, Ident: ident, Type: "integer", Value: integer},
			schema.Assignment{Kind: schema.AssignmentKind, HintId: placeholder, Ident: ident, Value: ident},
			schema.If{Kind: schema.IfKind, HintId: placeholder, Condition: boolean, Then: block(), Else: &other},
			schema.If{Kind: schema.IfKind, HintId: placeholder, Condition: boolean, Then: block()},
			schema.While{Kind: schema.WhileKind, HintId: placeholder, Condition: boolean, Body: block(schema.Break{Kind: schema.BreakKind, HintId: placeholder})},
			schema.For{Kind: schema.ForKind, HintId: placeholder, Ident: ident, Start: integer, End: integer, Body: block()},
			schema.Call{Kind: schema.CallKind, HintId: placeholder, Func: "rotate", Args: []schema.Expr{
				schema.Call{Kind: schema.CallKind, HintId: placeholder, Func: "identity", Args: []schema.Expr{
					schema.RotationLiteral{Kind: "literal", HintId: placeholder, Type: "rotation", Value: "left"},
				}},
			}},
			schema.DistanceTargetLiteral{Kind: "literal", HintId: placeholder, Type: "distance_target", Value: "goal"},
			schema.Return{Kind: schema.ReturnKind, HintId: placeholder, Value: schema.UnitLiteral{Kind: "literal", HintId: placeholder, Type: "unit"}},
		),
	}
	program := schema.Version0{Picolang: schema.Version0Picolang{V0: schema.Version0PicolangV0{
		Program: schema.PicolangV0Program{"hintId": fn, "kind": fn},
	}}}
	source, err := json.Marshal(program)
	if err != nil {
		t.Fatal(err)
	}
	ids := regexp.MustCompile(`"hintId"\s*:\s*"([^"]+)"`)
	wantCount := len(ids.FindAllSubmatch(source, -1))
	seen := make(map[string]struct{})
	for range 2 {
		got, err := assignGeneratedHintIDs(source)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := parse.Parse(got); err != nil {
			t.Errorf("reassigned program failed syntax validation: %v", err)
		}
		matches := ids.FindAllSubmatch(got, -1)
		if len(matches) != wantCount {
			t.Errorf("hint ID count = %d, want %d", len(matches), wantCount)
		}
		for _, match := range matches {
			id := string(match[1])
			parsed, err := uuid.Parse(id)
			if err != nil || parsed.Version() != 4 || parsed.Variant() != uuid.RFC4122 || id == string(placeholder) {
				t.Errorf("invalid server-generated ID: %q", id)
			}
			if _, exists := seen[id]; exists {
				t.Errorf("reused hint ID: %q", id)
			}
			seen[id] = struct{}{}
		}
		var want, actual schema.Version0
		if err := json.Unmarshal(source, &want); err != nil {
			t.Fatal(err)
		}
		masked := ids.ReplaceAll(got, []byte(`"hintId":"`+placeholder+`"`))
		if err := json.Unmarshal(masked, &actual); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(actual, want) {
			t.Errorf("reassignment changed fields other than hintId: %s", got)
		}
	}
}

func TestGeneratedHintIDsRejectInvalidSchemaNodes(t *testing.T) {
	t.Parallel()
	const source = `{"picolangVersion":0,"picolang":{"v0":{"program":{"main":{"kind":"function","hintId":"00000000-0000-4000-8000-000000000000","params":[],"returnType":"unit","body":{"kind":"block","hintId":"00000000-0000-4000-8000-000000000000","stmts":[{"kind":"return","hintId":"00000000-0000-4000-8000-000000000000","value":{"kind":"literal","hintId":"00000000-0000-4000-8000-000000000000","type":"integer","value":1}}]}}}}}}`
	for _, tt := range []struct{ name, from, to string }{
		{"unknown statement", `"kind":"return"`, `"kind":"unknown"`},
		{"wrong integer type", `"value":1`, `"value":"1"`},
		{"missing literal value", `,"value":1`, ``},
		{"null integer", `"value":1`, `"value":null`},
		{"unknown literal field", `"value":1`, `"value":1,"unexpected":true`},
		{"statement used as expression", `"kind":"literal"`, `"kind":"return"`},
		{"wrong version", `"picolangVersion":0`, `"picolangVersion":1`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := assignGeneratedHintIDs(jsontext.Value(strings.Replace(source, tt.from, tt.to, 1)))
			if !errors.Is(err, usecase.ErrInvalidGeneratedProgram) || got != nil {
				t.Errorf("got %s, %v; want an invalid generated program", got, err)
			}
		})
	}
}
