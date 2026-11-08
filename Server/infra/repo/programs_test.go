package repo

import (
	"context"
	"encoding/json/jsontext"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

func TestProgramsCreate(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	mustCreateUser(t, pool, "owner", nil)
	store := NewPrograms(pool)
	ctx := context.Background()
	for _, p := range []domain.Program{
		{ID: "battle", GoogleSubject: "owner", Name: "battle draft", Source: jsontext.Value(`{"integer":9223372036854775807}`), IsBattle: true},
		{ID: "tutorial", GoogleSubject: "owner", Name: "tutorial draft", Source: jsontext.Value(`{}`), TutorialID: ptr("tutorial-1")},
	} {
		if err := store.Create(ctx, p); err != nil {
			t.Errorf("Create: %v", err)
			continue
		}
		var got domain.Program
		if err := pool.QueryRow(ctx, "SELECT id, google_subject, name, source, is_battle, tutorial_id FROM programs WHERE id=$1", p.ID).Scan(&got.ID, &got.GoogleSubject, &got.Name, &got.Source, &got.IsBattle, &got.TutorialID); err != nil {
			t.Errorf("read stored program: %v", err)
			continue
		}
		if got.GoogleSubject != p.GoogleSubject || got.Name != p.Name || got.IsBattle != p.IsBattle || describeName(got.TutorialID) != describeName(p.TutorialID) {
			t.Errorf("stored = %+v, want %+v", got, p)
		}
		if err := got.Source.Compact(); err != nil {
			t.Errorf("compact source: %v", err)
		}
		if string(got.Source) != string(p.Source) {
			t.Errorf("source = %s, want %s", got.Source, p.Source)
		}
		if err := store.Create(ctx, p); err == nil {
			t.Error("duplicate ID was accepted")
		}
	}
	if err := store.Create(ctx, domain.Program{ID: "orphan", GoogleSubject: "missing", Name: "draft", Source: jsontext.Value(`{}`), IsBattle: true}); err == nil {
		t.Error("missing owner was accepted")
	}
}
