package matchrun

import (
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
)

func testLayouts(t *testing.T) *Layouts {
	t.Helper()
	layouts, err := NewLayouts()
	if err != nil {
		t.Fatal(err)
	}
	return layouts
}

func TestLayoutsKeepCachedDefinitionsIndependent(t *testing.T) {
	t.Parallel()
	repository := testLayouts(t)
	want, err := repository.List(t.Context())
	if err != nil || len(want) == 0 {
		t.Fatalf("load layouts: %v, count=%d", err, len(want))
	}
	first, err := repository.List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	first[0].Terrain[0][0] = schema.TerrainSea
	first[0].Luggage[0].X++
	first[0].Territories[0][0].X++
	first[0].Territories[1][0].Y++
	first[0].Robots[0][0].Position.X++
	first[0].FieldID = "modified"
	got, err := repository.List(t.Context())
	if err != nil {
		t.Errorf("list after mutation: %v", err)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Error("modifying a returned layout changed the cached definitions or another result")
	}
}

func TestLayoutsHaveDistinctFieldIDs(t *testing.T) {
	t.Parallel()
	layouts, err := testLayouts(t).List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]struct{})
	for _, layout := range layouts {
		if layout.FieldID == "" {
			t.Error("empty field ID")
		}
		if _, exists := seen[layout.FieldID]; exists {
			t.Errorf("duplicate field ID %q", layout.FieldID)
		}
		seen[layout.FieldID] = struct{}{}
	}
}
