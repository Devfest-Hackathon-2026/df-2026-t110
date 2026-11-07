// Package parse validates versioned picolang JSON and builds a named AST.
package parse

import (
	_ "embed"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"sync"

	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/google/jsonschema-go/jsonschema"
)

//go:generate go run ./internal/genschema

//go:embed schema.gen.json
var schemaJSON []byte

var resolvedSchema = sync.OnceValues(func() (*jsonschema.Resolved, error) {
	var schema jsonschema.Schema
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return nil, err
	}
	return schema.Resolve(nil)
})

var exactNumbers = json.UnmarshalFromFunc(func(decoder *jsontext.Decoder, value *any) error {
	if decoder.PeekKind() != '0' {
		return errors.ErrUnsupported
	}
	raw, err := decoder.ReadValue()
	if err != nil {
		return err
	}
	integer, err := parseInteger(string(raw))
	if err != nil {
		return err
	}
	*value = integer
	return nil
})

var _ bot.ParseFunc = Parse

// Parse accepts the versioned program JSON, not the enclosing API request body.
// It rejects duplicate object keys, validates the source schema and hint IDs,
// and preserves signed 64-bit integers without conversion through float64.
// Name resolution and type checking remain separate stages. On error no partial
// AST is returned. Calls are independent and may run concurrently.
func Parse(source []byte) (bot.ParsedProgram, error) {
	var value any
	if err := json.Unmarshal(source, &value, json.WithUnmarshalers(exactNumbers)); err != nil {
		return bot.ParsedProgram{}, fmt.Errorf("parse picolang JSON: %w", err)
	}
	schema, err := resolvedSchema()
	if err != nil {
		return bot.ParsedProgram{}, fmt.Errorf("resolve picolang schema: %w", err)
	}
	if err := schema.Validate(value); err != nil {
		return bot.ParsedProgram{}, fmt.Errorf("invalid picolang syntax: %w", err)
	}
	// The schema guarantees the object shapes used by the AST conversion below.
	root := value.(map[string]any)
	program := root["picolang"].(map[string]any)["v0"].(map[string]any)["program"].(map[string]any)
	p := parser{hints: make(map[string]struct{})}
	result := bot.ParsedProgram{Functions: make(map[bot.Name]bot.Function[bot.Name, bot.Untyped], len(program))}
	for _, name := range slices.Sorted(maps.Keys(program)) {
		p.function = name
		result.Functions[bot.Name(name)] = p.parseFunction(program[name].(map[string]any))
	}
	if p.err != nil {
		return bot.ParsedProgram{}, p.err
	}
	return result, nil
}

// All numeric positions in picolang v0 require integers. Accept decimal and
// exponent notation when mathematically integral, as JSON Schema does. Bound
// exponents before using big.Rat so short inputs cannot request huge powers.
func parseInteger(text string) (int64, error) {
	if integer, err := strconv.ParseInt(text, 10, 64); err == nil {
		return integer, nil
	}
	mantissa, exponent, hasExponent := strings.Cut(strings.ToLower(text), "e")
	if strings.Trim(mantissa, "-0.") == "" {
		return 0, nil
	}
	if hasExponent {
		n, err := strconv.ParseInt(exponent, 10, 64)
		if err != nil || n > int64(len(text))+20 || n < -int64(len(text))-20 {
			return 0, fmt.Errorf("number %q must be an integer in the signed 64-bit range", text)
		}
	}
	value, ok := new(big.Rat).SetString(text)
	if !ok || !value.IsInt() || !value.Num().IsInt64() {
		return 0, fmt.Errorf("number %q must be an integer in the signed 64-bit range", text)
	}
	return value.Num().Int64(), nil
}

type parser struct {
	hints    map[string]struct{}
	function string
	err      error
}

func (p *parser) source(node map[string]any) bot.Source {
	hint := node["hintId"].(string)
	key := strings.ToLower(hint)
	if _, exists := p.hints[key]; exists && p.err == nil {
		p.err = fmt.Errorf("function %q: duplicate hintId %q", p.function, hint)
	}
	p.hints[key] = struct{}{}
	return bot.Source{HintID: bot.HintID(hint)}
}

func (p *parser) parseFunction(node map[string]any) bot.Function[bot.Name, bot.Untyped] {
	result := bot.Function[bot.Name, bot.Untyped]{Source: p.source(node), ReturnType: p.parseType(node["returnType"])}
	params := node["params"].([]any)
	result.Params = make([]bot.Param[bot.Name], len(params))
	for i, raw := range params {
		param := raw.(map[string]any)
		result.Params[i] = bot.Param[bot.Name]{Source: p.source(param), Ident: p.ident(param["ident"].(map[string]any)), Type: p.parseType(param["type"])}
	}
	result.Body = p.block(node["body"].(map[string]any))
	return result
}

func (p *parser) parseType(value any) bot.Type {
	if vector, ok := value.(map[string]any); ok {
		return bot.VectorType{ElementType: p.parseType(vector["elementType"])}
	}
	switch value.(string) {
	case "bool":
		return bot.BoolType{}
	case "integer":
		return bot.IntegerType{}
	case "unit":
		return bot.UnitType{}
	case "terrain":
		return bot.TerrainType{}
	case "rotation":
		return bot.RotationType{}
	case "distance_target":
		return bot.DistanceTargetType{}
	default:
		p.err = fmt.Errorf("unsupported type %q", value)
		return nil
	}
}

func (p *parser) ident(node map[string]any) bot.Ident[bot.Name] {
	return bot.Ident[bot.Name]{Source: p.source(node), Variable: bot.Name(node["name"].(string))}
}

func (p *parser) block(node map[string]any) bot.Block[bot.Name, bot.Untyped] {
	result := bot.Block[bot.Name, bot.Untyped]{Source: p.source(node)}
	stmts := node["stmts"].([]any)
	result.Stmts = make([]bot.Stmt[bot.Name, bot.Untyped], len(stmts))
	for i, stmt := range stmts {
		result.Stmts[i] = p.stmt(stmt.(map[string]any))
	}
	return result
}

func (p *parser) stmt(node map[string]any) bot.Stmt[bot.Name, bot.Untyped] {
	kind := node["kind"].(string)
	if kind == "literal" || kind == "ident" || kind == "call" {
		return p.expr(node)
	}
	source := p.source(node)
	switch kind {
	case "declare":
		return bot.Declaration[bot.Name, bot.Untyped]{Source: source, Type: p.parseType(node["type"]), Ident: p.ident(node["ident"].(map[string]any)), Value: p.expr(node["value"].(map[string]any))}
	case "assign":
		return bot.Assignment[bot.Name, bot.Untyped]{Source: source, Ident: p.ident(node["ident"].(map[string]any)), Value: p.expr(node["value"].(map[string]any))}
	case "if":
		result := bot.If[bot.Name, bot.Untyped]{Source: source, Condition: p.expr(node["condition"].(map[string]any)), Then: p.block(node["then"].(map[string]any))}
		if other, exists := node["else"]; exists {
			block := p.block(other.(map[string]any))
			result.Else = &block
		}
		return result
	case "while":
		return bot.While[bot.Name, bot.Untyped]{Source: source, Condition: p.expr(node["condition"].(map[string]any)), Body: p.block(node["body"].(map[string]any))}
	case "for":
		return bot.For[bot.Name, bot.Untyped]{Source: source, Ident: p.ident(node["ident"].(map[string]any)), Start: p.expr(node["start"].(map[string]any)), End: p.expr(node["end"].(map[string]any)), Body: p.block(node["body"].(map[string]any))}
	case "return":
		return bot.Return[bot.Name, bot.Untyped]{Source: source, Value: p.expr(node["value"].(map[string]any))}
	case "break":
		return bot.Break[bot.Name, bot.Untyped]{Source: source}
	case "continue":
		return bot.Continue[bot.Name, bot.Untyped]{Source: source}
	default:
		p.err = fmt.Errorf("unsupported statement kind %q", kind)
		return nil
	}
}

func (p *parser) expr(node map[string]any) bot.Expr[bot.Name, bot.Untyped] {
	switch kind := node["kind"].(string); kind {
	case "ident":
		return bot.IdentExpr[bot.Name, bot.Untyped]{Ident: p.ident(node)}
	case "call":
		result := bot.CallExpr[bot.Name, bot.Untyped]{Source: p.source(node), Func: bot.Name(node["func"].(string))}
		args := node["args"].([]any)
		result.Args = make([]bot.Expr[bot.Name, bot.Untyped], len(args))
		for i, arg := range args {
			result.Args[i] = p.expr(arg.(map[string]any))
		}
		return result
	case "literal":
		result := bot.LiteralExpr[bot.Name, bot.Untyped]{Source: p.source(node)}
		switch node["type"].(string) {
		case "bool":
			result.Value = bot.BoolLiteral{Value: node["value"].(bool)}
		case "integer":
			result.Value = bot.IntegerLiteral{Value: node["value"].(int64)}
		case "unit":
			result.Value = bot.UnitLiteral{}
		case "rotation":
			result.Value = bot.RotationLiteral{Value: values.Rotation(node["value"].(string))}
		case "distance_target":
			result.Value = bot.DistanceTargetLiteral{Value: values.DistanceTarget(node["value"].(string))}
		default:
			p.err = fmt.Errorf("unsupported literal type %q", node["type"])
		}
		return result
	default:
		p.err = fmt.Errorf("unsupported expression kind %q", kind)
		return nil
	}
}
