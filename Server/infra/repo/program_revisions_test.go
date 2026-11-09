package repo

import (
	"encoding/json/jsontext"
	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"testing"
)

func TestProgramRevisions(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	mustCreateUser(t, pool, "owner", nil)
	programs := NewPrograms(pool)
	if err := programs.Create(t.Context(), domain.Program{ID: "p", GoogleSubject: "owner", Name: "first", Source: jsontext.Value(`{"value":9223372036854775807}`), IsBattle: true}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"second", "second"} {
		if _, err := pool.Exec(t.Context(), `UPDATE programs SET name=$1 WHERE id='p'`, name); err != nil {
			t.Fatal(err)
		}
	}
	got, err := programs.Get(t.Context(), "p")
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if got.Revision != 3 || got.Name != "second" {
		t.Errorf("latest = %+v", got)
	}
	var count int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM program_revisions WHERE program_id='p'`).Scan(&count); err != nil {
		t.Errorf("read revision count: %v", err)
	} else if count != 3 {
		t.Errorf("revisions = %d", count)
	}
	var name, source string
	if err := pool.QueryRow(t.Context(), `SELECT name,source FROM program_revisions WHERE program_id='p' AND revision=1`).Scan(&name, &source); err != nil {
		t.Errorf("read archived revision: %v", err)
	} else if name != "first" {
		t.Errorf("old name = %s", name)
	}
	for _, query := range []string{`UPDATE program_revisions SET name='changed'`, `DELETE FROM program_revisions`, `UPDATE programs SET is_battle=false,tutorial_id='lesson' WHERE id='p'`} {
		if _, err := pool.Exec(t.Context(), query); err == nil {
			t.Errorf("accepted %s", query)
		}
	}
	list, err := programs.GetByTutorialID(t.Context(), "owner", "lesson")
	if err != nil || len(list) != 0 {
		t.Errorf("purpose changed: %v %v", list, err)
	}
}
