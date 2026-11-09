package main

import (
	json "encoding/json/v2"
	"fmt"
	"math/rand/v2"

	"github.com/dvpico/Programming-Learning-AR/Server/game/robot"
)

// botProfiles returns the supported behaviors in the default generation order.
func botProfiles() []string { return []string{"sentry", "wanderer", "courier", "miner"} }

type botProgram struct {
	Profile    string `json:"profile"`
	Seed       uint64 `json:"seed"`
	Delay      int    `json:"delay"`
	TurnChance int    `json:"turnChance"`
	Source     []byte `json:"-"`
}

// generateBot emits the same source for the same profile and seed, including hint IDs.
// Every result is checked by the production parser, indexer and type checker.
func generateBot(profile string, seed uint64) (botProgram, error) {
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)) //nolint:gosec // Opponent behavior is intentionally reproducible.
	f := builder{seed: seed}
	p := botProgram{Profile: profile, Seed: seed, Delay: rng.IntN(8), TurnChance: rng.IntN(16) + 5}
	rotation := "left"
	if rng.IntN(2) == 0 {
		rotation = "right"
	}
	turn := func() any { return f.call("rotate", f.literal("rotation", rotation)) }
	about := func() []any { return []any{turn(), turn()} }
	var body []any
	switch profile {
	case "sentry":
		body = []any{f.call("wait", f.literal("integer", rng.IntN(4)+1))}
	case "wanderer", "courier", "miner":
		// Carrying is tracked only after a successful pick_up/release, including conflicts.
		pickupCondition := f.call("and", f.call("is_luggage", f.ident("front")), f.call("not", f.call("is_own_territory", f.ident("front"))))
		pickupTurn := f.node("if", "condition", f.ident("carrying"), "then", f.block(about()...),
			"else", f.block(f.call("wait", f.call("random_integer", f.literal("integer", 1), f.literal("integer", 3)))))
		acquire := f.node("if", "condition", pickupCondition,
			"then", f.block(f.assign("carrying", f.call("pick_up")), pickupTurn),
			"else", f.block(f.call("walk")))
		deliveryCondition := f.call("and", f.call("is_own_territory", f.ident("front")), f.call("not", f.call("is_luggage", f.ident("front"))))
		deliveryTurn := f.node("if", "condition", f.call("not", f.ident("carrying")), "then", f.block(about()...))
		deliver := f.node("if", "condition", deliveryCondition,
			"then", f.block(f.assign("carrying", f.call("not", f.call("release"))), deliveryTurn),
			"else", f.block(f.call("walk")))
		body = []any{
			f.node("declare", "type", "terrain", "ident", f.ident("front"), "value", f.call("search_front")),
			f.node("if", "condition", f.ident("carrying"), "then", f.block(deliver), "else", f.block(acquire)),
		}
	default:
		return botProgram{}, fmt.Errorf("unknown profile %q (choose sentry, wanderer, courier, miner)", profile)
	}
	stmts := []any{f.call("wait", f.literal("integer", p.Delay))}
	if profile != "sentry" {
		stmts = append(stmts, f.node("declare", "type", "bool", "ident", f.ident("carrying"), "value", f.literal("bool", false)))
	}
	stmts = append(stmts, f.node("while", "condition", f.literal("bool", true), "body", f.block(body...)), f.ret())
	functions := map[string]any{"main": f.function(stmts...)}
	if profile != "sentry" {
		// Sea is never a release target. Only miners spend drill durability.
		walk := []any{}
		if profile == "wanderer" || profile == "miner" {
			walk = append(walk, f.node("if", "condition", f.call("lt", f.call("random_integer", f.literal("integer", 0), f.literal("integer", 99)), f.literal("integer", p.TurnChance)), "then", f.block(turn(), f.ret())))
		}
		if profile == "miner" {
			walk = append(walk, f.node("if", "condition", f.call("and", f.call("is_rock", f.call("search_front")), f.call("gt", f.call("drill_durability"), f.literal("integer", 0))), "then", f.block(f.call("drill"))))
		}
		walk = append(walk,
			f.node("if", "condition", f.call("is_sea", f.call("search_front")), "then", f.block(about()...),
				"else", f.block(f.node("if", "condition", f.call("not", f.call("forward")), "then", f.block(turn())))),
			f.ret())
		functions["walk"] = f.function(walk...)
	}
	data, err := json.Marshal(map[string]any{"picolangVersion": 0, "picolang": map[string]any{"v0": map[string]any{"program": functions}}}, json.Deterministic(true))
	if err != nil {
		return botProgram{}, err
	}
	bot, err := robot.NewPicolangBot(data, seed, defaultLimits(), false)
	if err != nil {
		return botProgram{}, fmt.Errorf("validate generated %s: %w", profile, err)
	}
	if err := bot.Close(); err != nil {
		return botProgram{}, err
	}
	p.Source = data
	return p, nil
}

type builder struct {
	seed uint64
	next uint64
}

func (f *builder) node(kind string, fields ...any) map[string]any {
	f.next++
	v := map[string]any{"kind": kind, "hintId": fmt.Sprintf("%08x-%04x-4000-8000-%012x", f.seed>>32, f.seed&0xffff, f.next)}
	for i := 0; i+1 < len(fields); i += 2 {
		v[fields[i].(string)] = fields[i+1]
	}
	return v
}
func (f *builder) literal(typ string, value any) any {
	return f.node("literal", "type", typ, "value", value)
}
func (f *builder) ident(name string) any { return f.node("ident", "name", name) }
func (f *builder) call(name string, args ...any) any {
	if args == nil {
		args = []any{}
	}
	return f.node("call", "func", name, "args", args)
}
func (f *builder) block(stmts ...any) any {
	if stmts == nil {
		stmts = []any{}
	}
	return f.node("block", "stmts", stmts)
}
func (f *builder) assign(name string, value any) any {
	return f.node("assign", "ident", f.ident(name), "value", value)
}
func (f *builder) ret() any { return f.node("return", "value", f.literal("unit", nil)) }
func (f *builder) function(stmts ...any) any {
	return f.node("function", "params", []any{}, "returnType", "unit", "body", f.block(stmts...))
}
