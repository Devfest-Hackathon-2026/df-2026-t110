package repo

import (
	"context"
	"reflect"
	"testing"
)

func TestBattleProgramsSet(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	mustCreateUser(t, pool, "owner", nil)
	ctx := context.Background()
	for _, id := range []string{"a", "b", "c"} {
		if _, err := pool.Exec(ctx, `INSERT INTO programs (id, google_subject, name, source, is_battle) VALUES ($1,'owner','battle','{}',true)`, id); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewBattlePrograms(pool)
	for _, ids := range [][]string{{"a", "b", "c"}, {"c", "a", "c"}} {
		if err := repo.Set(ctx, "owner", ids); err != nil {
			t.Fatal(err)
		}
		var a, b, c string
		if err := pool.QueryRow(ctx, `SELECT program_1,program_2,program_3 FROM battle_programs WHERE google_subject='owner'`).Scan(&a, &b, &c); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual([]string{a, b, c}, ids) {
			t.Errorf("saved %v, want %v", []string{a, b, c}, ids)
		}
	}
}
