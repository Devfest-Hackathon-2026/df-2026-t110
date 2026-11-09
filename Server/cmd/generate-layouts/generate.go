package main

import (
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
)

const (
	width          = 14
	height         = 11
	layoutCount    = 64
	luggagePerSide = 5
	rocksPerSide   = 12
	maxAttempts    = 100_000
)

func generate(seed uint64) ([]domain.BattleLayout, error) {
	// A reproducible offline catalog needs a seeded PRNG, not cryptographic randomness.
	rng := rand.New(rand.NewPCG(seed, 1)) //nolint:gosec // Used only for offline board generation.
	layouts := make([]domain.BattleLayout, 0, layoutCount)
	seenTerrain := make(map[string]struct{})
	seenLuggage := make(map[string]struct{})
	for range maxAttempts {
		layout := candidate(rng)
		if !hasAlternateRoutes(layout.Terrain) {
			continue
		}
		terrainKey, luggageKey := signatures(layout)
		if _, exists := seenTerrain[terrainKey]; exists {
			continue
		}
		if _, exists := seenLuggage[luggageKey]; exists {
			continue
		}
		seenTerrain[terrainKey] = struct{}{}
		seenLuggage[luggageKey] = struct{}{}
		layout.ID = fmt.Sprintf("battle-1-v2-seed-%d-layout-%02d", seed, len(layouts)+1)
		layouts = append(layouts, layout)
		if len(layouts) == layoutCount {
			return layouts, nil
		}
	}
	return nil, fmt.Errorf("generated only %d layouts in %d attempts", len(layouts), maxAttempts)
}

func signatures(layout domain.BattleLayout) (string, string) {
	positions := slices.Clone(layout.Luggage)
	slices.SortFunc(positions, func(a, b schema.Position) int {
		if a.Y != b.Y {
			return a.Y - b.Y
		}
		return a.X - b.X
	})
	return fmt.Sprint(layout.Terrain), fmt.Sprint(positions)
}

func candidate(rng *rand.Rand) domain.BattleLayout {
	layout := domain.BattleLayout{FieldID: "battle-1", DrillLimit: 2, Turns: 160, LuggagePoints: 1, DiscardPenalty: 1}
	var near, central []schema.Position
	for y := range height {
		row := make([]schema.Terrain, width)
		for x := range row {
			row[x] = schema.TerrainLand
		}
		layout.Terrain = append(layout.Terrain, row)
		layout.Territories[0] = append(layout.Territories[0], schema.Position{X: 0, Y: y})
		layout.Territories[1] = append(layout.Territories[1], schema.Position{X: width - 1, Y: y})
		for x := 1; x < width/2; x++ {
			p := schema.Position{X: x, Y: y}
			if x <= 3 {
				near = append(near, p)
			} else {
				central = append(central, p)
			}
		}
	}
	rng.Shuffle(len(near), func(i, j int) { near[i], near[j] = near[j], near[i] })
	rng.Shuffle(len(central), func(i, j int) { central[i], central[j] = central[j], central[i] })
	// Two luggage positions close to home, three closer to the center.
	leftLuggage := append(slices.Clone(near[:2]), central[:luggagePerSide-2]...)
	for _, p := range leftLuggage {
		layout.Luggage = append(layout.Luggage, p, schema.Position{X: width - 1 - p.X, Y: p.Y})
	}
	available := append(slices.Clone(near[2:]), central[luggagePerSide-2:]...)
	rng.Shuffle(len(available), func(i, j int) { available[i], available[j] = available[j], available[i] })
	for _, p := range available[:rocksPerSide] {
		layout.Terrain[p.Y][p.X] = schema.TerrainRock
		layout.Terrain[p.Y][width-1-p.X] = schema.TerrainRock
	}
	for slot := range domain.BattleRobotCount {
		y := height/2 + slot
		layout.Robots[0][slot] = schema.FieldDefinitionRobot{Position: schema.Position{X: 0, Y: y}, Direction: "east"}
		layout.Robots[1][slot] = schema.FieldDefinitionRobot{Position: schema.Position{X: width - 1, Y: y}, Direction: "west"}
	}
	return layout
}

// hasAlternateRoutes rejects disconnected land and single-cell bottlenecks
// using a depth-first search for articulation points. Robots and luggage do
// not block this terrain check; robots can still temporarily block a path.
func hasAlternateRoutes(terrain [][]schema.Terrain) bool {
	land := make(map[schema.Position]struct{})
	var start schema.Position
	for y, row := range terrain {
		for x, cell := range row {
			if cell == schema.TerrainLand {
				start = schema.Position{X: x, Y: y}
				land[start] = struct{}{}
			}
		}
	}
	if len(land) < 3 {
		return false
	}
	discovered := make(map[schema.Position]int)
	low := make(map[schema.Position]int)
	bottleneck := false
	var visit func(schema.Position, schema.Position)
	visit = func(p, parent schema.Position) {
		discovered[p] = len(discovered) + 1
		low[p] = discovered[p]
		children := 0
		for _, n := range neighbors(p) {
			if _, exists := land[n]; !exists {
				continue
			}
			if discovered[n] == 0 {
				children++
				visit(n, p)
				low[p] = min(low[p], low[n])
				if p != parent && low[n] >= discovered[p] {
					bottleneck = true
				}
			} else if n != parent {
				low[p] = min(low[p], discovered[n])
			}
		}
		if p == parent && children > 1 {
			bottleneck = true
		}
	}
	visit(start, start)
	return len(discovered) == len(land) && !bottleneck
}

func neighbors(p schema.Position) [4]schema.Position {
	return [4]schema.Position{{X: p.X + 1, Y: p.Y}, {X: p.X - 1, Y: p.Y}, {X: p.X, Y: p.Y + 1}, {X: p.X, Y: p.Y - 1}}
}
