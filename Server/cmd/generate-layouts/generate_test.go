package main

import (
	"bytes"
	json "encoding/json/v2"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
)

func testCatalog(t *testing.T, seed uint64) []domain.BattleLayout {
	t.Helper()
	layouts, err := generate(seed)
	if err != nil {
		t.Fatal(err)
	}
	return layouts
}

func TestGenerateReproducibleCatalog(t *testing.T) {
	t.Parallel()
	first := testCatalog(t, 165)
	if !reflect.DeepEqual(first, testCatalog(t, 165)) {
		t.Error("same seed produced different catalogs")
	}
	other := testCatalog(t, 166)
	fieldIDs := make(map[string]struct{})
	for _, layout := range first {
		fieldIDs[layout.FieldID] = struct{}{}
	}
	for _, layout := range other {
		if _, exists := fieldIDs[layout.FieldID]; exists {
			t.Errorf("different seeds reused field ID %q", layout.FieldID)
		}
	}
	if reflect.DeepEqual(first[0].Terrain, other[0].Terrain) || reflect.DeepEqual(first[0].Luggage, other[0].Luggage) {
		t.Error("changing the seed did not change both terrain and luggage")
	}
}

func TestGenerateLayoutConstraints(t *testing.T) {
	t.Parallel()
	layouts := testCatalog(t, 165)
	if len(layouts) != 64 {
		t.Errorf("catalog size=%d, want 64", len(layouts))
	}
	fieldIDs := make(map[string]struct{})
	terrainKeys := make(map[string]struct{})
	luggageKeys := make(map[string]struct{})
	for _, layout := range layouts {
		if _, exists := fieldIDs[layout.FieldID]; exists {
			t.Errorf("duplicate field ID %q", layout.FieldID)
		}
		fieldIDs[layout.FieldID] = struct{}{}
		if layout.FieldID == "" {
			t.Error("field ID must not be empty")
		}
		terrainKey, luggageKey := signatures(layout)
		if _, exists := terrainKeys[terrainKey]; exists {
			t.Errorf("repeated terrain in %s", layout.FieldID)
		}
		terrainKeys[terrainKey] = struct{}{}
		if _, exists := luggageKeys[luggageKey]; exists {
			t.Errorf("repeated luggage in %s", layout.FieldID)
		}
		luggageKeys[luggageKey] = struct{}{}
		t.Run(layout.FieldID, func(t *testing.T) {
			t.Parallel()
			if len(layout.Terrain) != 11 {
				t.Fatalf("height=%d, want 11", len(layout.Terrain))
			}
			rocks := 0
			for y, row := range layout.Terrain {
				if len(row) != 14 {
					t.Fatalf("row %d width=%d, want 14", y, len(row))
				}
				for x, cell := range row {
					if cell != row[13-x] {
						t.Errorf("terrain not mirrored at (%d,%d)", x, y)
					}
					switch cell {
					case schema.TerrainRock:
						rocks++
					case schema.TerrainLand:
					default:
						t.Errorf("invalid terrain %q at (%d,%d)", cell, x, y)
					}
				}
			}
			if rocks != 24 {
				t.Errorf("rocks=%d, want 24", rocks)
			}
			positions := make(map[schema.Position]struct{})
			near, central := 0, 0
			for _, p := range layout.Luggage {
				if _, exists := positions[p]; exists {
					t.Errorf("duplicate luggage at %+v", p)
				}
				positions[p] = struct{}{}
				if p.X < 1 || p.X >= 13 || p.Y < 0 || p.Y >= 11 {
					t.Errorf("luggage outside neutral land: %+v", p)
					continue
				}
				if layout.Terrain[p.Y][p.X] != schema.TerrainLand {
					t.Errorf("luggage on rock: %+v", p)
				}
				if p.X <= 3 {
					near++
				} else if p.X <= 6 {
					central++
				}
			}
			if len(layout.Luggage) != 10 || near != 2 || central != 3 {
				t.Errorf("luggage=%d, near=%d, central=%d, want 10/2/3", len(layout.Luggage), near, central)
			}
			for p := range positions {
				if _, exists := positions[schema.Position{X: 13 - p.X, Y: p.Y}]; !exists {
					t.Errorf("luggage not mirrored at %+v", p)
				}
			}
			for team, x := range []int{0, 13} {
				var territory []schema.Position
				for y := range 11 {
					territory = append(territory, schema.Position{X: x, Y: y})
					if layout.Terrain[y][x] != schema.TerrainLand {
						t.Errorf("territory (%d,%d) is not land", x, y)
					}
				}
				if !reflect.DeepEqual(layout.Territories[team], territory) {
					t.Errorf("incorrect territory for team %d", team)
				}
				direction := "east"
				if team == 1 {
					direction = "west"
				}
				for slot, robot := range layout.Robots[team] {
					if robot.Position != (schema.Position{X: x, Y: 5 + slot}) || string(robot.Direction) != direction {
						t.Errorf("team %d robot %d start=%+v", team, slot, robot)
					}
				}
			}
			if layout.DrillLimit != 2 || layout.Turns != 160 || layout.LuggagePoints != 1 || layout.DiscardPenalty != 1 {
				t.Error("incorrect battle rules")
			}
		})
	}
}

func TestGeneratedLandHasAlternateRoutes(t *testing.T) {
	t.Parallel()
	for _, layout := range testCatalog(t, 165) {
		t.Run(layout.FieldID, func(t *testing.T) {
			t.Parallel()
			land := make(map[schema.Position]struct{})
			for y, row := range layout.Terrain {
				for x, cell := range row {
					if cell == schema.TerrainLand {
						land[schema.Position{X: x, Y: y}] = struct{}{}
					}
				}
			}
			// Independently verify the articulation-point algorithm by blocking
			// every land cell in turn and flood-filling all remaining land.
			for blocked := range land {
				var start schema.Position
				for p := range land {
					if p != blocked {
						start = p
						break
					}
				}
				visited := map[schema.Position]struct{}{start: {}}
				queue := []schema.Position{start}
				for i := 0; i < len(queue); i++ {
					p := queue[i]
					for _, d := range []schema.Position{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
						n := schema.Position{X: p.X + d.X, Y: p.Y + d.Y}
						_, isLand := land[n]
						_, seen := visited[n]
						if isLand && !seen && n != blocked {
							visited[n] = struct{}{}
							queue = append(queue, n)
						}
					}
				}
				if len(visited) != len(land)-1 {
					t.Errorf("blocking %+v disconnected land", blocked)
				}
			}
		})
	}
}

func TestRejectDisconnectedLandAndBottlenecks(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		rows []string
		want bool
	}{
		{"empty", nil, false},
		{"single cell", []string{"."}, false},
		{"disconnected", []string{".#.", ".#."}, false},
		{"bottleneck", []string{".#.", "...", ".#."}, false},
		{"cycle", []string{"...", ".#.", "..."}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var terrain [][]schema.Terrain
			for _, row := range tt.rows {
				var cells []schema.Terrain
				for _, cell := range row {
					value := schema.TerrainLand
					if cell == '#' {
						value = schema.TerrainRock
					}
					cells = append(cells, value)
				}
				terrain = append(terrain, cells)
			}
			if got := hasAlternateRoutes(terrain); got != tt.want {
				t.Errorf("hasAlternateRoutes=%v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunWritesCatalog(t *testing.T) {
	t.Parallel()
	output := filepath.Join(t.TempDir(), "nested", "layouts.json")
	if err := run([]string{"--output", output, "--seed", "166"}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output) //nolint:gosec // The path is inside this test's temporary directory.
	if err != nil {
		t.Fatal(err)
	}
	var got []domain.BattleLayout
	if err := json.Unmarshal(data, &got, json.RejectUnknownMembers(true)); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, testCatalog(t, 166)) {
		t.Error("output did not match generated catalog")
	}
	if err := run([]string{"--output", output, "--seed", "166"}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(output) //nolint:gosec // The path is inside this test's temporary directory.
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, again) {
		t.Error("repeated invocation produced different JSON bytes")
	}
}

func TestRunRejectsInvalidArgumentsAndHandlesHelp(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{nil, {"--unknown"}, {"--seed", "bad"}, {"--output", "unused.json", "extra"}} {
		if err := run(args, &bytes.Buffer{}); err == nil {
			t.Errorf("accepted invalid arguments %v", args)
		}
	}
	if err := run([]string{"--help"}, &bytes.Buffer{}); !errors.Is(err, flag.ErrHelp) {
		t.Errorf("help error=%v", err)
	}
}
