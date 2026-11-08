package repo

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

func TestProgramsCreate(t *testing.T) {
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
		var got domain.Program
		if err := pool.QueryRow(ctx, "SELECT id, google_subject, name, source, is_battle, tutorial_id FROM programs WHERE id=$1", p.ID).Scan(&got.ID, &got.GoogleSubject, &got.Name, &got.Source, &got.IsBattle, &got.TutorialID); err != nil {
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
	if err := programs.Create(ctx, domain.Program{ID: "orphan", GoogleSubject: "missing", Name: "draft", Source: jsontext.Value(`{}`), IsBattle: true}); err == nil {
		t.Error("missing owner was accepted")
	}
}

func TestProgramsGet(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	mustCreateUser(t, pool, "owner", nil)
	programs := NewPrograms(pool)
	for _, want := range []domain.Program{
		{ID: "battle", GoogleSubject: "owner", Name: "battle draft", Source: jsontext.Value(`{"integer":9223372036854775807}`), IsBattle: true},
		{ID: "tutorial", GoogleSubject: "owner", Name: "tutorial draft", Source: jsontext.Value(`{}`), TutorialID: ptr("tutorial-1")},
	} {
		if _, err := pool.Exec(t.Context(), "INSERT INTO programs (id, google_subject, name, source, is_battle, tutorial_id) VALUES ($1,$2,$3,$4,$5,$6)", want.ID, want.GoogleSubject, want.Name, []byte(want.Source), want.IsBattle, want.TutorialID); err != nil {
			t.Fatal(err)
		}
		got, err := programs.Get(t.Context(), want.ID)
		if err != nil {
			t.Errorf("Get: %v", err)
			continue
		}
		if got.ID != want.ID || got.GoogleSubject != want.GoogleSubject || got.Name != want.Name || got.IsBattle != want.IsBattle || describeName(got.TutorialID) != describeName(want.TutorialID) {
			t.Errorf("Get = %+v, want %+v", got, want)
		}
		if err := got.Source.Compact(); err != nil {
			t.Errorf("compact source: %v", err)
		}
		if string(got.Source) != string(want.Source) {
			t.Errorf("source = %s, want %s", got.Source, want.Source)
		}
	}
	if _, err := programs.Get(t.Context(), "missing"); !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("Get(missing) = %v, want ErrNotFound", err)
	}
}
