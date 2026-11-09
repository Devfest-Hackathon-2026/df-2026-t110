package router

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/parse"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/google/uuid"
)

// Decode the schema-generated models before collecting their hint ID fields.
// Stmt and Expr are generated as interfaces, so decode their concrete variants
// explicitly instead of traversing maps or guessing which objects are nodes.
func assignGeneratedHintIDs(source jsontext.Value) (jsontext.Value, error) {
	// Generated decoders do not enforce every JSON Schema constraint (for
	// example additionalProperties). Check syntax before decoding can drop data.
	if err := parse.ValidateSyntax(source); err != nil {
		return nil, fmt.Errorf("%w: %w", usecase.ErrInvalidGeneratedProgram, err)
	}
	var program schema.Version0
	if err := json.Unmarshal(source, &program); err != nil {
		return nil, fmt.Errorf("%w: %w", usecase.ErrInvalidGeneratedProgram, err)
	}
	functions := make(map[string]*schema.PicolangFunctionDefinition)
	var hints []*schema.HintId
	for name, fn := range program.Picolang.V0.Program {
		functions[name] = &fn
		hints = append(hints, &fn.HintId)
		for i := range fn.Params {
			hints = append(hints, &fn.Params[i].HintId, &fn.Params[i].Ident.HintId)
		}
		if err := decodeGeneratedBlock(&fn.Body, &hints); err != nil {
			return nil, fmt.Errorf("%w: function %q: %w", usecase.ErrInvalidGeneratedProgram, name, err)
		}
	}
	// All nodes have passed their generated decoders before any ID is assigned.
	for _, hint := range hints {
		id, err := uuid.NewRandom()
		if err != nil {
			return nil, fmt.Errorf("generate hint ID: %w", err)
		}
		*hint = schema.HintId(id.String())
	}
	for name, fn := range functions {
		program.Picolang.V0.Program[name] = *fn
	}
	return json.Marshal(program)
}

func decodeGeneratedBlock(block *schema.Block, hints *[]*schema.HintId) error {
	*hints = append(*hints, &block.HintId)
	for i, stmt := range block.Stmts {
		node, err := decodeGeneratedNode(stmt, false, hints)
		if err != nil {
			return err
		}
		block.Stmts[i] = node
	}
	return nil
}

func decodeGeneratedNode(value any, expression bool, hints *[]*schema.HintId) (any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var discriminator struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(data, &discriminator); err != nil {
		return nil, err
	}
	var node any
	switch discriminator.Kind {
	case schema.IdentKind:
		node = new(schema.Ident)
	case schema.CallKind:
		node = new(schema.Call)
	case schema.BoolLiteralKind:
		node = new(schema.Literal)
	default:
		if expression {
			return nil, fmt.Errorf("invalid expression kind %q", discriminator.Kind)
		}
		switch discriminator.Kind {
		case schema.DeclarationKind:
			node = new(schema.Declaration)
		case schema.AssignmentKind:
			node = new(schema.Assignment)
		case schema.IfKind:
			node = new(schema.If)
		case schema.WhileKind:
			node = new(schema.While)
		case schema.ForKind:
			node = new(schema.For)
		case schema.ReturnKind:
			node = new(schema.Return)
		case schema.BreakKind:
			node = new(schema.Break)
		case schema.ContinueKind:
			node = new(schema.Continue)
		default:
			return nil, fmt.Errorf("unknown statement kind %q", discriminator.Kind)
		}
	}
	if err := json.Unmarshal(data, node); err != nil {
		return nil, err
	}

	var hint *schema.HintId
	var expressions []*schema.Expr
	var blocks []*schema.Block
	switch n := node.(type) {
	case *schema.Ident:
		hint = &n.HintId
	case *schema.Call:
		hint = &n.HintId
		for i := range n.Args {
			expressions = append(expressions, &n.Args[i])
		}
	case *schema.Literal:
		switch {
		case n.Bool != nil:
			hint = &n.Bool.HintId
		case n.Integer != nil:
			hint = &n.Integer.HintId
		case n.Unit != nil:
			hint = &n.Unit.HintId
		case n.Rotation != nil:
			hint = &n.Rotation.HintId
		case n.DistanceTarget != nil:
			hint = &n.DistanceTarget.HintId
		default:
			return nil, fmt.Errorf("unsupported literal variant")
		}
	case *schema.Declaration:
		hint, expressions = &n.HintId, []*schema.Expr{&n.Value}
		*hints = append(*hints, &n.Ident.HintId)
	case *schema.Assignment:
		hint, expressions = &n.HintId, []*schema.Expr{&n.Value}
		*hints = append(*hints, &n.Ident.HintId)
	case *schema.If:
		hint, expressions, blocks = &n.HintId, []*schema.Expr{&n.Condition}, []*schema.Block{&n.Then}
		if n.Else != nil {
			blocks = append(blocks, n.Else)
		}
	case *schema.While:
		hint, expressions, blocks = &n.HintId, []*schema.Expr{&n.Condition}, []*schema.Block{&n.Body}
	case *schema.For:
		hint, expressions, blocks = &n.HintId, []*schema.Expr{&n.Start, &n.End}, []*schema.Block{&n.Body}
		*hints = append(*hints, &n.Ident.HintId)
	case *schema.Return:
		hint, expressions = &n.HintId, []*schema.Expr{&n.Value}
	case *schema.Break:
		hint = &n.HintId
	case *schema.Continue:
		hint = &n.HintId
	default:
		return nil, fmt.Errorf("unsupported generated node %T", node)
	}
	*hints = append(*hints, hint)
	for _, expr := range expressions {
		decoded, err := decodeGeneratedNode(*expr, true, hints)
		if err != nil {
			return nil, err
		}
		*expr = decoded
	}
	for _, block := range blocks {
		if err := decodeGeneratedBlock(block, hints); err != nil {
			return nil, err
		}
	}
	return node, nil
}
