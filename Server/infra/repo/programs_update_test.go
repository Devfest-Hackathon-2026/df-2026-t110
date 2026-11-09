package repo

import (
	"encoding/json/jsontext"
	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"testing"
)

func TestProgramsUpdate(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	mustCreateUser(t, pool, "owner", nil)
	r := NewPrograms(pool)
	if err := r.Create(t.Context(), domain.Program{ID: "p", GoogleSubject: "owner", Name: "old", Source: jsontext.Value(`{}`), IsBattle: true}); err != nil {
		t.Fatal(err)
	}
	got, err := r.Update(t.Context(), domain.Program{ID: "p", GoogleSubject: "owner", Name: "new", Source: jsontext.Value(`{"integer":9223372036854775807}`)})
	if err != nil {
		t.Fatal(err)
	}
	if got.Revision != 2 || !got.IsBattle || got.Name != "new" {
		t.Errorf("updated: %+v", got)
	}
}
