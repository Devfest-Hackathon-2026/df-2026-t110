package repo

import (
	"encoding/json/jsontext"
	"fmt"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/google/uuid"
)

func TestMatchProgramReferencesBulkInsert(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	mustCreateUser(t, pool, "owner", ptr("Owner"))
	mustCreateUser(t, pool, "opponent", ptr("Opponent"))
	programs := NewPrograms(pool)
	for i := range 2 {
		if err := programs.Create(t.Context(), domain.Program{ID: fmt.Sprintf("p%d", i), GoogleSubject: "owner", Name: "original", Source: jsontext.Value(`{}`), IsBattle: true}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(t.Context(), `UPDATE programs SET name='edited' WHERE id='p1'`); err != nil {
		t.Fatal(err)
	}
	teams := [2][]domain.ProgramRef{
		{{ID: "p1", Revision: 2}, {ID: "p0", Revision: 1}, {ID: "p1", Revision: 1}},
		{{ID: "p0", Revision: 1}, {ID: "p1", Revision: 2}, {ID: "p0", Revision: 1}},
	}
	record := domain.MatchRecord{ID: uuid.Must(uuid.NewV7()).String(), GoogleSubject: "owner", OpponentSubject: "opponent", OpponentName: "Opponent", Outcome: domain.MatchOutcome("win"), Result: jsontext.Value(`{}`), ChallengerPrograms: teams[0], OpponentPrograms: teams[1], RatingDelta: 10}
	// A failure in the last row must roll back the entire completion, including the run result.
	record.OpponentPrograms[2].Revision = 99
	if _, err := completeMatch(t.Context(), pool, record); err == nil {
		t.Error("accepted nonexistent final revision")
	}
	var remaining int
	if err := pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM runs WHERE result IS NOT NULL) + (SELECT count(*) FROM matches) + (SELECT count(*) FROM run_programs)`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Errorf("failed bulk insert left %d rows", remaining)
	}
	if got := getRating(t, pool, "owner"); got != domain.InitialRating {
		t.Errorf("failed match changed rating: %d", got)
	}
	record.OpponentPrograms[2].Revision = 1
	runID, err := completeMatch(t.Context(), pool, record)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := pool.Query(t.Context(), `SELECT side,robot_index,program_id,revision FROM run_programs WHERE run_id=$1 ORDER BY side,robot_index`, runID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var side, index int
		var id string
		var revision int
		if err := rows.Scan(&side, &index, &id, &revision); err != nil {
			t.Fatal(err)
		}
		expectedSide, expectedIndex := count/3, count%3
		if count >= 6 {
			t.Error("unexpected additional row")
			continue
		}
		want := teams[expectedSide][expectedIndex]
		if side != expectedSide || index != expectedIndex || id != want.ID || revision != want.Revision {
			t.Errorf("slot %d: (%d,%d,%s,%d), want (%d,%d,%s,%d)", count, side, index, id, revision, expectedSide, expectedIndex, want.ID, want.Revision)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		t.Errorf("read references: %v", err)
	}
	if count != 6 {
		t.Errorf("references=%d, want 6", count)
	}
}
