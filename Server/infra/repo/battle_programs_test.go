package repo

import (
	"context"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
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
	for _, ids := range [][]string{{"a", "b", "c"}, {"c", "a", "c"}, {"b", "a"}, {"a", "b", "a", "c"}, {}} {
		if err := repo.Set(ctx, "owner", ids); err != nil {
			t.Errorf("set %v: %v", ids, err)
			continue
		}
		assertBattlePrograms(t, pool, ids)
	}
	if err := repo.Set(ctx, "owner", []string{"a", "b", "a"}); err != nil {
		t.Errorf("set rollback baseline: %v", err)
		return
	}
	if err := repo.Set(ctx, "owner", []string{"a", "missing"}); err == nil {
		t.Error("missing program accepted")
	}
	assertBattlePrograms(t, pool, []string{"a", "b", "a"})
}

func assertBattlePrograms(t *testing.T, pool *pgxpool.Pool, want []string) {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT program_id, robot_index FROM battle_programs WHERE google_subject='owner' ORDER BY robot_index`)
	if err != nil {
		t.Errorf("read slots: %v", err)
		return
	}
	defer rows.Close()
	got := []string{}
	for rows.Next() {
		var id string
		var index int
		if err := rows.Scan(&id, &index); err != nil {
			t.Errorf("scan slot: %v", err)
			return
		}
		if index != len(got) {
			t.Errorf("index = %d, want %d", index, len(got))
		}
		got = append(got, id)
	}
	if err := rows.Err(); err != nil {
		t.Errorf("read slots: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("saved %v, want %v", got, want)
	}
}
