package memory_test

import (
	"math"
	"reflect"
	"testing"

	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
)

func add(t *testing.T, s *memory.Store, value memory.Value) memory.ID {
	t.Helper()
	id, err := s.Add(value)
	if err != nil {
		t.Fatalf("Add(%#v): %v", value, err)
	}
	return id
}

func release(t *testing.T, s *memory.Store, id memory.ID) {
	t.Helper()
	if err := s.Release(id); err != nil {
		t.Errorf("Release(%d): %v", id, err)
	}
}

func assertCount(t *testing.T, s *memory.Store, id memory.ID, want uint64) {
	t.Helper()
	if got, exists := s.RefCount(id); !exists || got != want {
		t.Errorf("RefCount(%d) = (%d, %t), want (%d, true)", id, got, exists, want)
	}
}

func assertValue(t *testing.T, s *memory.Store, id memory.ID, want memory.Value) {
	t.Helper()
	if got, exists := s.Get(id); !exists || !reflect.DeepEqual(got, want) {
		t.Errorf("Get(%d) = (%#v, %t), want (%#v, true)", id, got, exists, want)
	}
}

func assertFreed(t *testing.T, s *memory.Store, ids ...memory.ID) {
	t.Helper()
	for _, id := range ids {
		if got, exists := s.Get(id); exists || got != nil {
			t.Errorf("Get(%d) = (%#v, %t) after release", id, got, exists)
		}
		if got, exists := s.RefCount(id); exists || got != 0 {
			t.Errorf("RefCount(%d) = (%d, %t) after release", id, got, exists)
		}
	}
}

func TestPrimitiveLifetimes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value memory.Value
	}{
		{"true", memory.Bool{Value: true}},
		{"false", memory.Bool{}},
		{"minimum integer", memory.Integer{Value: math.MinInt64}},
		{"maximum integer", memory.Integer{Value: math.MaxInt64}},
		{"unit", memory.Unit{}},
		{"plain", memory.Terrain{Value: values.LandTerrain{Kind: values.PlainLand{}}}},
		{"goal", memory.Terrain{Value: values.LandTerrain{Kind: values.GoalLand{}}}},
		{"checkpoint", memory.Terrain{Value: values.LandTerrain{Kind: values.CheckpointLand{Order: 3}}}},
		{"luggage", memory.Terrain{Value: values.LandTerrain{Kind: values.PlainLand{}, HasLuggage: true}}},
		{"own territory with luggage", memory.Terrain{Value: values.LandTerrain{Kind: values.PlainLand{}, HasLuggage: true, Territory: values.OwnTerritory}}},
		{"enemy territory", memory.Terrain{Value: values.LandTerrain{Kind: values.PlainLand{}, Territory: values.EnemyTerritory}}},
		{"rock", memory.Terrain{Value: values.RockTerrain{}}},
		{"sea", memory.Terrain{Value: values.SeaTerrain{}}},
		{"left", memory.Rotation{Value: values.Left}},
		{"right", memory.Rotation{Value: values.Right}},
		{"distance robot", memory.DistanceTarget{Value: values.DistanceRobot}},
		{"distance ally", memory.DistanceTarget{Value: values.DistanceAlly}},
		{"distance enemy", memory.DistanceTarget{Value: values.DistanceEnemy}},
		{"distance rock", memory.DistanceTarget{Value: values.DistanceRock}},
		{"distance luggage", memory.DistanceTarget{Value: values.DistanceLuggage}},
		{"distance sea", memory.DistanceTarget{Value: values.DistanceSea}},
		{"distance goal", memory.DistanceTarget{Value: values.DistanceGoal}},
		{"distance territory", memory.DistanceTarget{Value: values.DistanceTerritory}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var s memory.Store
			id := add(t, &s, tc.value)
			assertCount(t, &s, id, 1)
			assertValue(t, &s, id, tc.value)
			assertCount(t, &s, id, 1) // Reading does not retain.
			if err := s.Retain(id); err != nil {
				t.Errorf("Retain(%d): %v", id, err)
			}
			assertCount(t, &s, id, 2)
			release(t, &s, id)
			assertCount(t, &s, id, 1)
			assertValue(t, &s, id, tc.value)
			release(t, &s, id)
			assertFreed(t, &s, id)
			assertCount(t, &s, memory.RootID, 1)
		})
	}
}

func TestStoreOwnsCommonRoot(t *testing.T) {
	t.Parallel()
	var s memory.Store
	rootBytes := s.MemoryUsage()
	assertValue(t, &s, memory.RootID, memory.ListRoot{})
	assertCount(t, &s, memory.RootID, 1)
	if err := s.Release(memory.RootID); err == nil {
		t.Error("Release accepted the store-owned root reference")
	}
	assertCount(t, &s, memory.RootID, 1)
	a := add(t, &s, memory.ListRoot{})
	b := add(t, &s, memory.ListRoot{})
	if a != memory.RootID || b != a {
		t.Errorf("roots = %d, %d; want common ID %d", a, b, memory.RootID)
	}
	assertCount(t, &s, memory.RootID, 3)
	assertUsage(t, &s, rootBytes)
	release(t, &s, a)
	release(t, &s, b)
	assertValue(t, &s, memory.RootID, memory.ListRoot{})
	assertCount(t, &s, memory.RootID, 1)
	assertUsage(t, &s, rootBytes)
}

func TestSharedAndNestedLists(t *testing.T) {
	t.Parallel()
	var s memory.Store
	rootBytes := s.MemoryUsage()
	primitiveBytes, nodeBytes := rootBytes+8, rootBytes+16
	a := add(t, &s, memory.Integer{Value: 1})
	b := add(t, &s, memory.Integer{Value: 2})
	prefix := add(t, &s, memory.ListNode{Prev: memory.RootID, Value: a})
	left := add(t, &s, memory.ListNode{Prev: prefix, Value: b})
	right := add(t, &s, memory.ListNode{Prev: prefix, Value: b})
	assertCount(t, &s, memory.RootID, 2)
	assertCount(t, &s, a, 2)
	assertCount(t, &s, b, 3)
	assertCount(t, &s, prefix, 3)
	assertUsage(t, &s, rootBytes+2*primitiveBytes+3*nodeBytes)
	// Only the two branch heads retain the list and its elements now.
	release(t, &s, a)
	release(t, &s, b)
	release(t, &s, prefix)
	release(t, &s, left)
	assertFreed(t, &s, left)
	assertCount(t, &s, prefix, 1)
	assertCount(t, &s, a, 1)
	assertCount(t, &s, b, 1)
	assertUsage(t, &s, rootBytes+2*primitiveBytes+2*nodeBytes)
	assertValue(t, &s, prefix, memory.ListNode{Prev: memory.RootID, Value: a})
	assertValue(t, &s, right, memory.ListNode{Prev: prefix, Value: b})
	nested := add(t, &s, memory.ListNode{Prev: memory.RootID, Value: right})
	assertCount(t, &s, right, 2)
	assertCount(t, &s, memory.RootID, 3)
	release(t, &s, right)
	assertValue(t, &s, nested, memory.ListNode{Prev: memory.RootID, Value: right})
	assertCount(t, &s, right, 1)
	assertUsage(t, &s, rootBytes+2*primitiveBytes+3*nodeBytes)
	release(t, &s, nested)
	assertFreed(t, &s, a, b, prefix, left, right, nested)
	assertCount(t, &s, memory.RootID, 1)
	assertUsage(t, &s, rootBytes)
}

func TestBothNodeEdgesRetainSameValue(t *testing.T) {
	t.Parallel()
	var s memory.Store
	empty := add(t, &s, memory.ListNode{Prev: memory.RootID, Value: memory.RootID})
	assertCount(t, &s, memory.RootID, 3)
	nested := add(t, &s, memory.ListNode{Prev: empty, Value: empty})
	assertCount(t, &s, empty, 3)
	release(t, &s, empty)
	assertCount(t, &s, empty, 2)
	release(t, &s, nested)
	assertFreed(t, &s, empty, nested)
	assertCount(t, &s, memory.RootID, 1)
}

func TestStoredPayloadsAreImmutable(t *testing.T) {
	t.Parallel()
	var s memory.Store
	terrain := memory.Terrain{Value: values.LandTerrain{Kind: values.CheckpointLand{Order: 2}}}
	valueID := add(t, &s, terrain)
	terrain.Value = values.SeaTerrain{}
	got, _ := s.Get(valueID)
	observation, ok := got.(memory.Terrain)
	if !ok {
		t.Fatalf("terrain has unexpected representation %T", got)
	}
	observation.Value = values.RockTerrain{}
	assertValue(t, &s, valueID, memory.Terrain{Value: values.LandTerrain{Kind: values.CheckpointLand{Order: 2}}})
	node := memory.ListNode{Prev: memory.RootID, Value: valueID}
	nodeID := add(t, &s, node)
	node.Prev, node.Value = nodeID, nodeID
	got, _ = s.Get(nodeID)
	copy, ok := got.(memory.ListNode)
	if !ok {
		t.Fatalf("list node has unexpected representation %T", got)
	}
	copy.Prev, copy.Value = nodeID, nodeID
	assertValue(t, &s, nodeID, memory.ListNode{Prev: memory.RootID, Value: valueID})
	assertCount(t, &s, memory.RootID, 2)
	assertCount(t, &s, valueID, 2)
}

func TestTerrainObservationsDoNotRetainReferenceMarkers(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"land", "rock", "sea"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			marker := &values.LandTerrain{Kind: values.GoalLand{}}
			var input, want values.TerrainValue
			switch name {
			case "land":
				input = values.LandTerrain{Kind: values.PlainLand{}, HasLuggage: true, Territory: values.OwnTerritory, TerrainValue: marker}
				want = values.LandTerrain{Kind: values.PlainLand{}, HasLuggage: true, Territory: values.OwnTerritory}
			case "rock":
				input, want = values.RockTerrain{TerrainValue: marker}, values.RockTerrain{}
			case "sea":
				input, want = values.SeaTerrain{TerrainValue: marker}, values.SeaTerrain{}
			}
			var s memory.Store
			base := s.MemoryUsage()
			id := add(t, &s, memory.Terrain{Value: input})
			marker.Kind, marker.HasLuggage = values.CheckpointLand{Order: 7}, true
			assertValue(t, &s, id, memory.Terrain{Value: want})
			release(t, &s, id)
			assertUsage(t, &s, base)
		})
	}
}

func TestRejectInvalidPayloads(t *testing.T) {
	t.Parallel()
	var nilBool *memory.Bool
	for _, tc := range []struct {
		name  string
		value memory.Value
	}{
		{"nil", nil},
		{"typed nil", nilBool},
		{"bool pointer", &memory.Bool{}},
		{"integer pointer", &memory.Integer{}},
		{"unit pointer", &memory.Unit{}},
		{"terrain pointer", &memory.Terrain{Value: values.SeaTerrain{}}},
		{"rotation pointer", &memory.Rotation{Value: values.Left}},
		{"distance target pointer", &memory.DistanceTarget{Value: values.DistanceRobot}},
		{"root pointer", &memory.ListRoot{}},
		{"node pointer", &memory.ListNode{}},
		{"nil terrain", memory.Terrain{}},
		{"terrain observation pointer", memory.Terrain{Value: &values.SeaTerrain{}}},
		{"missing land kind", memory.Terrain{Value: values.LandTerrain{}}},
		{"land kind pointer", memory.Terrain{Value: values.LandTerrain{Kind: &values.CheckpointLand{Order: 1}}}},
		{"invalid rotation", memory.Rotation{Value: "back"}},
		{"missing distance target", memory.DistanceTarget{}},
		{"unknown distance target", memory.DistanceTarget{Value: "invalid"}},
		{"rotation as distance target", memory.DistanceTarget{Value: "left"}},
		{"unknown territory", memory.Terrain{Value: values.LandTerrain{Kind: values.PlainLand{}, Territory: "invalid"}}},
		{"goal as territory", memory.Terrain{Value: values.LandTerrain{Kind: values.GoalLand{}, Territory: values.OwnTerritory}}},
		{"checkpoint as territory", memory.Terrain{Value: values.LandTerrain{Kind: values.CheckpointLand{Order: 1}, Territory: values.EnemyTerritory}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var s memory.Store
			before := s.MemoryUsage()
			if _, err := s.Add(tc.value); err == nil {
				t.Errorf("Add accepted invalid value %#v", tc.value)
			}
			assertCount(t, &s, memory.RootID, 1)
			assertUsage(t, &s, before)
		})
	}
}

func TestRejectInvalidReferencesWithoutRetaining(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		node func(value, freed memory.ID) memory.ListNode
	}{
		{"unknown previous", func(value, _ memory.ID) memory.ListNode { return memory.ListNode{Prev: 999, Value: value} }},
		{"primitive previous", func(value, _ memory.ID) memory.ListNode { return memory.ListNode{Prev: value, Value: value} }},
		{"unknown value", func(_, _ memory.ID) memory.ListNode { return memory.ListNode{Prev: memory.RootID, Value: 999} }},
		{"freed previous", func(value, freed memory.ID) memory.ListNode { return memory.ListNode{Prev: freed, Value: value} }},
		{"freed value", func(_, freed memory.ID) memory.ListNode { return memory.ListNode{Prev: memory.RootID, Value: freed} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var s memory.Store
			value := add(t, &s, memory.Unit{})
			freed := add(t, &s, memory.ListNode{Prev: memory.RootID, Value: value})
			release(t, &s, freed)
			before := s.MemoryUsage()
			if _, err := s.Add(tc.node(value, freed)); err == nil {
				t.Error("Add accepted invalid references")
			}
			assertCount(t, &s, memory.RootID, 1)
			assertCount(t, &s, value, 1)
			assertFreed(t, &s, freed)
			assertUsage(t, &s, before)
		})
	}
}

func TestRejectUnknownAndReleasedIDs(t *testing.T) {
	t.Parallel()
	var s memory.Store
	freed := add(t, &s, memory.Unit{})
	release(t, &s, freed)
	next := add(t, &s, memory.Unit{})
	if next == freed {
		t.Error("Add reused a freed ID")
	}
	before := s.MemoryUsage()
	for _, id := range []memory.ID{freed, 999, math.MaxUint64} {
		assertFreed(t, &s, id)
		if err := s.Retain(id); err == nil {
			t.Errorf("Retain accepted invalid ID %d", id)
		}
		if err := s.Release(id); err == nil {
			t.Errorf("Release accepted invalid ID %d", id)
		}
	}
	assertCount(t, &s, next, 1)
	assertCount(t, &s, memory.RootID, 1)
	assertUsage(t, &s, before)
}

func TestReleaseLongList(t *testing.T) {
	t.Parallel()
	var s memory.Store
	rootBytes := s.MemoryUsage()
	value := add(t, &s, memory.Unit{})
	tail := add(t, &s, memory.ListRoot{})
	const length = 10000
	for range length {
		next := add(t, &s, memory.ListNode{Prev: tail, Value: value})
		release(t, &s, tail)
		tail = next
	}
	assertCount(t, &s, value, length+1)
	assertUsage(t, &s, 2*rootBytes+length*(rootBytes+16))
	release(t, &s, value)
	release(t, &s, tail)
	for id := memory.ID(1); id <= tail; id++ {
		assertFreed(t, &s, id)
	}
	assertCount(t, &s, memory.RootID, 1)
	assertUsage(t, &s, rootBytes)
}
