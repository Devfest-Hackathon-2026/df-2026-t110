package memory_test

import (
	"strconv"
	"testing"

	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
)

func assertUsage(t *testing.T, s *memory.Store, want uint64) {
	t.Helper()
	if got := s.MemoryUsage(); got != want {
		t.Errorf("MemoryUsage() = %d, want %d bytes", got, want)
	}
}

func TestMemoryUsagePerEntry(t *testing.T) {
	t.Parallel()
	wordBytes := uint64(strconv.IntSize / 8)
	for _, tc := range []struct {
		name         string
		value        memory.Value
		payloadBytes uint64
	}{
		{"bool", memory.Bool{Value: true}, 1},
		{"integer", memory.Integer{Value: 42}, 8},
		{"unit", memory.Unit{}, 0},
		{"none", memory.None{}, 2 * wordBytes},
		{"some", memory.Some{Element: memory.RootID}, 2*wordBytes + 8},
		{"plain", memory.Terrain{Value: values.LandTerrain{Kind: values.PlainLand{}}}, 9 * wordBytes},
		{"goal", memory.Terrain{Value: values.LandTerrain{Kind: values.GoalLand{}}}, 9 * wordBytes},
		{"checkpoint", memory.Terrain{Value: values.LandTerrain{Kind: values.CheckpointLand{Order: 3}}}, 9*wordBytes + 8},
		{"luggage", memory.Terrain{Value: values.LandTerrain{Kind: values.PlainLand{}, HasLuggage: true}}, 9 * wordBytes},
		{"own territory with luggage", memory.Terrain{Value: values.LandTerrain{Kind: values.PlainLand{}, HasLuggage: true, Territory: values.OwnTerritory}}, 9*wordBytes + 3},
		{"enemy territory", memory.Terrain{Value: values.LandTerrain{Kind: values.PlainLand{}, Territory: values.EnemyTerritory}}, 9*wordBytes + 5},
		{"rock", memory.Terrain{Value: values.RockTerrain{}}, 4 * wordBytes},
		{"sea", memory.Terrain{Value: values.SeaTerrain{}}, 4 * wordBytes},
		{"left", memory.Rotation{Value: values.Left}, 2*wordBytes + 4},
		{"right", memory.Rotation{Value: values.Right}, 2*wordBytes + 5},
		{"distance robot", memory.DistanceTarget{Value: values.DistanceRobot}, 2*wordBytes + 5},
		{"distance ally", memory.DistanceTarget{Value: values.DistanceAlly}, 2*wordBytes + 4},
		{"distance enemy", memory.DistanceTarget{Value: values.DistanceEnemy}, 2*wordBytes + 5},
		{"distance rock", memory.DistanceTarget{Value: values.DistanceRock}, 2*wordBytes + 4},
		{"distance luggage", memory.DistanceTarget{Value: values.DistanceLuggage}, 2*wordBytes + 7},
		{"distance sea", memory.DistanceTarget{Value: values.DistanceSea}, 2*wordBytes + 3},
		{"distance goal", memory.DistanceTarget{Value: values.DistanceGoal}, 2*wordBytes + 4},
		{"distance territory", memory.DistanceTarget{Value: values.DistanceTerritory}, 2*wordBytes + 9},
		{"list node", memory.ListNode{Prev: memory.RootID, Value: memory.RootID}, 16},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var s memory.Store
			rootBytes := s.MemoryUsage()
			// The empty root has no payload; all of its usage is entry metadata.
			if rootBytes == 0 {
				t.Error("usage does not include the store-owned root metadata")
			}
			id := add(t, &s, tc.value)
			want := 2*rootBytes + tc.payloadBytes
			assertUsage(t, &s, want)
			if err := s.Retain(id); err != nil {
				t.Errorf("Retain(%d): %v", id, err)
			}
			s.Get(id)
			s.RefCount(id)
			assertUsage(t, &s, want)
			release(t, &s, id)
			assertUsage(t, &s, want)
			release(t, &s, id)
			assertUsage(t, &s, rootBytes)
		})
	}
}

func TestMemoryUsageCountsDistinctIDs(t *testing.T) {
	t.Parallel()
	var s memory.Store
	rootBytes := s.MemoryUsage()
	a := add(t, &s, memory.Integer{Value: 1})
	b := add(t, &s, memory.Integer{Value: 1})
	assertUsage(t, &s, rootBytes+2*(rootBytes+8))
	release(t, &s, a)
	assertUsage(t, &s, 2*rootBytes+8)
	release(t, &s, b)
	assertUsage(t, &s, rootBytes)
}

func TestPeakMemoryUsageSurvivesReleases(t *testing.T) {
	t.Parallel()
	var s memory.Store
	rootBytes := s.MemoryUsage()
	if got := s.PeakMemoryUsage(); got != rootBytes {
		t.Errorf("initial peak = %d, want %d", got, rootBytes)
	}
	a := add(t, &s, memory.Integer{Value: 1})
	b := add(t, &s, memory.Integer{Value: 2})
	peak := s.MemoryUsage()
	release(t, &s, a)
	release(t, &s, b)
	c := add(t, &s, memory.Unit{})
	release(t, &s, c)
	if got := s.PeakMemoryUsage(); got != peak {
		t.Errorf("peak after releases = %d, want %d", got, peak)
	}
	assertUsage(t, &s, rootBytes)
}
