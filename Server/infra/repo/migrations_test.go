package repo

import (
	"os"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRatingsMigration(t *testing.T) {
	t.Parallel()

	migrated := newTestPoolWithDDL(t, "testdata/schema_before_ratings.sql")
	if _, err := migrated.Exec(t.Context(), `
		INSERT INTO users (google_subject, name) VALUES ('owner', 'Owner'), ('opponent', 'Opponent');
		INSERT INTO runs (id, google_subject, field_id, result) VALUES
			('00000000-0000-7000-8000-000000000001', 'owner', 'field', '{"summary":{"score":{"player":2,"opponent":1}}}'),
			('00000000-0000-7000-8000-000000000002', 'owner', 'field', '{"summary":{"score":{"player":0,"opponent":1}}}'),
			('00000000-0000-7000-8000-000000000003', 'owner', 'field', '{"summary":{"score":{"player":1,"opponent":1}}}');
		INSERT INTO matches (id, run_id, opponent_google_subject, opponent_name, selected_layout_id)
		SELECT gen_random_uuid(), id, 'opponent', 'Opponent', 'layout' FROM runs`); err != nil {
		t.Fatalf("insert legacy matches: %v", err)
	}

	migration, err := os.ReadFile("sql/migrations/20261109_ratings.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := migrated.Exec(t.Context(), string(migration)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	rows, err := migrated.Query(t.Context(), `
		SELECT matches.outcome, matches.rating_before IS NULL
		FROM matches JOIN runs ON runs.id = matches.run_id
		ORDER BY runs.id`)
	if err != nil {
		t.Fatal(err)
	}

	type legacy struct {
		Outcome    string
		NullRating bool
	}
	var got []legacy
	for rows.Next() {
		var m legacy
		if err := rows.Scan(&m.Outcome, &m.NullRating); err != nil {
			t.Fatal(err)
		}
		got = append(got, m)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	if want := []legacy{{"win", true}, {"loss", true}, {"draw", true}}; !reflect.DeepEqual(got, want) {
		t.Errorf("legacy matches=%+v, want %+v", got, want)
	}

	if rating := getRating(t, migrated, "owner"); rating != 1000 {
		t.Errorf("existing user rating=%d, want 1000", rating)
	}

	fresh := newTestPool(t)
	fieldMigration, err := os.ReadFile("sql/migrations/20261110_field_id.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := migrated.Exec(t.Context(), string(fieldMigration)); err != nil {
		t.Fatalf("apply field ID migration: %v", err)
	}
	for _, table := range []string{"users", "matches"} {
		got, want := describeTable(t, migrated, table), describeTable(t, fresh, table)
		if len(want) == 0 {
			t.Fatalf("schema.sql has no %s table", table)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("migrated %s differs from schema.sql:\n got %q\nwant %q", table, got, want)
		}
	}
}

func TestFieldIDMigration(t *testing.T) {
	t.Parallel()
	pool := newTestPoolWithDDL(t, "testdata/schema_before_ratings.sql")
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO users (google_subject, name) VALUES ('owner', 'Owner'), ('opponent', 'Opponent');
		INSERT INTO runs (id, google_subject, field_id, result) VALUES
			('00000000-0000-7000-8000-000000000001', 'owner', 'battle-1', '{"history":["saved"]}'),
			('00000000-0000-7000-8000-000000000002', 'owner', 'first-step-1', '{"history":["tutorial"]}');
		INSERT INTO matches (id, run_id, opponent_google_subject, opponent_name, selected_layout_id)
		VALUES ('00000000-0000-7000-8000-000000000003', '00000000-0000-7000-8000-000000000001', 'opponent', 'Opponent', 'battle-1-layout-2')`); err != nil {
		t.Fatalf("insert historical records: %v", err)
	}
	migration, err := os.ReadFile("sql/migrations/20261110_field_id.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), string(migration)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	for _, tt := range []struct {
		id, field, history string
	}{
		{"00000000-0000-7000-8000-000000000001", "battle-1-layout-2", "saved"},
		{"00000000-0000-7000-8000-000000000002", "first-step-1", "tutorial"},
	} {
		var field, history string
		if err := pool.QueryRow(t.Context(), "SELECT field_id, result->'history'->>0 FROM runs WHERE id=$1", tt.id).Scan(&field, &history); err != nil {
			t.Errorf("read migrated run: %v", err)
			continue
		}
		if field != tt.field || history != tt.history {
			t.Errorf("run %s: field=%q history=%q, want %q/%q", tt.id, field, history, tt.field, tt.history)
		}
	}
	var columnCount int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM information_schema.columns
		WHERE table_schema=current_schema() AND table_name='matches' AND column_name='selected_layout_id'`).Scan(&columnCount); err != nil {
		t.Errorf("check removed column: %v", err)
	} else if columnCount != 0 {
		t.Error("selected_layout_id still exists")
	}
}

// describeTable lists columns and constraint definitions without constraint
// names, which differ between ADD CHECK and inline CHECK.
func describeTable(t *testing.T, pool *pgxpool.Pool, table string) []string {
	t.Helper()

	rows, err := pool.Query(t.Context(), `
		SELECT format('%s %s null=%s default=%s', column_name, data_type, is_nullable, coalesce(column_default, ''))
		FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = $1
		UNION ALL
		SELECT pg_get_constraintdef(oid)
		FROM pg_constraint
		WHERE conrelid = (quote_ident(current_schema()) || '.' || quote_ident($1))::regclass
		ORDER BY 1`, table)
	if err != nil {
		t.Fatalf("describe %s: %v", table, err)
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return lines
}
