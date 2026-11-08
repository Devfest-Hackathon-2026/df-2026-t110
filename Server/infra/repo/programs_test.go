package repo

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

func TestProgramsCreateAndGet(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	mustCreateUser(t, pool, "owner", nil)
	programs := NewPrograms(pool)
	ctx := context.Background()
	for _, p := range []domain.Program{
		{ID: "battle", GoogleSubject: "owner", Name: "battle draft", Source: jsontext.Value(`{"integer":9223372036854775807}`), IsBattle: true},
		{ID: "tutorial", GoogleSubject: "owner", Name: "tutorial draft", Source: jsontext.Value(`{}`), TutorialID: ptr("tutorial-1")},
	} {
		if err := programs.Create(ctx, p); err != nil {
			t.Errorf("Create: %v", err)
			continue
		}
		got, err := programs.Get(ctx, p.ID)
		if err != nil {
			t.Errorf("read stored program: %v", err)
			continue
		}
		if got.ID != p.ID || got.GoogleSubject != p.GoogleSubject || got.Name != p.Name || got.IsBattle != p.IsBattle || describeName(got.TutorialID) != describeName(p.TutorialID) {
			t.Errorf("stored = %+v, want %+v", got, p)
		}
		if err := got.Source.Compact(); err != nil {
			t.Errorf("compact source: %v", err)
		}
		if string(got.Source) != string(p.Source) {
			t.Errorf("source = %s, want %s", got.Source, p.Source)
		}
		if err := programs.Create(ctx, p); err == nil {
			t.Error("duplicate ID was accepted")
		}
	}
	if _, err := programs.Get(ctx, "missing"); !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("Get(missing) = %v, want ErrNotFound", err)
	}
	if err := programs.Create(ctx, domain.Program{ID: "orphan", GoogleSubject: "missing", Name: "draft", Source: jsontext.Value(`{}`), IsBattle: true}); err == nil {
		t.Error("missing owner was accepted")
	}
}
