package repo

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/matchrun"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"
)

func TestSelectBattleOpponent(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	programs, battle := NewPrograms(pool), NewBattlePrograms(pool)
	for _, subject := range []string{"owner", "opponent", "nameless", "partial", "extra", "tutorial", "foreign", "gap"} {
		var name *string
		if subject != "nameless" {
			name = ptr(subject)
		}
		mustCreateUser(t, pool, subject, name)
		p := domain.Program{ID: subject, GoogleSubject: subject, Name: subject, Source: jsontext.Value(`{}`), IsBattle: true}
		if subject == "tutorial" {
			p.IsBattle = false
			p.TutorialID = ptr("lesson")
		}
		if err := programs.Create(t.Context(), p); err != nil {
			t.Fatal(err)
		}
		ids := []string{subject, subject, subject}
		switch subject {
		case "partial":
			ids = ids[:2]
		case "extra":
			ids = append(ids, subject)
		case "foreign":
			ids[1] = "owner"
		}
		if err := battle.Set(t.Context(), subject, ids); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(t.Context(), "UPDATE battle_programs SET robot_index=3 WHERE google_subject='gap' AND robot_index=0"); err != nil {
		t.Fatal(err)
	}
	if err := programs.Create(t.Context(), domain.Program{ID: "opponent-b", GoogleSubject: "opponent", Name: "B", Source: jsontext.Value(`{"source":"B"}`), IsBattle: true}); err != nil {
		t.Fatal(err)
	}
	want := []string{"opponent-b", "opponent", "opponent-b"}
	if err := battle.Set(t.Context(), "opponent", want); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		team, err := NewMatches(pool).SelectOpponent(t.Context(), "owner")
		if err != nil {
			t.Errorf("SelectOpponent: %v", err)
			continue
		}
		var ids []string
		for _, p := range team.Programs {
			ids = append(ids, p.ID)
			if p.GoogleSubject != "opponent" || !p.IsBattle || !p.Source.IsValid() {
				t.Errorf("program=%+v", p)
			}
		}
		if team.User.GoogleSubject != "opponent" || team.User.Name == nil || *team.User.Name != "opponent" || !reflect.DeepEqual(ids, want) {
			t.Errorf("team=%+v ids=%v", team, ids)
		}
	}
	if err := battle.Set(t.Context(), "opponent", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := NewMatches(pool).SelectOpponent(t.Context(), "owner"); !errors.Is(err, usecase.ErrNoOpponent) {
		t.Errorf("no opponent error=%v", err)
	}
}

type matchSession struct{}

func (matchSession) CurrentUser(_ context.Context, id string) (domain.User, error) {
	return domain.User{GoogleSubject: id}, nil
}

func TestPostMatchPersistsAndReplays(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	source, err := os.ReadFile("../matchrun/testdata/query_move.json")
	if err != nil {
		t.Fatal(err)
	}
	programs := NewPrograms(pool)
	for _, subject := range []string{"owner", "opponent"} {
		mustCreateUser(t, pool, subject, ptr(subject))
		if err := programs.Create(t.Context(), domain.Program{ID: subject, GoogleSubject: subject, Name: "forward", Source: source, IsBattle: true}); err != nil {
			t.Fatal(err)
		}
	}
	if err := NewBattlePrograms(pool).Set(t.Context(), "opponent", []string{"opponent", "opponent", "opponent"}); err != nil {
		t.Fatal(err)
	}
	matches := NewMatches(pool)
	layouts, err := matchrun.NewLayouts()
	if err != nil {
		t.Fatal(err)
	}
	server := echo.New()
	api.RegisterHandlersWithOptions(server, router.NewRouter(router.Config{Matches: usecase.NewMatches(usecase.MatchDeps{
		Users: NewUsers(pool), Programs: programs, Opponents: matches, Layouts: layouts, Executor: matchrun.NewExecutor(), Matches: matches,
	})}), api.RegisterHandlersOptions{BaseURL: "/api", OperationMiddlewares: map[string][]echo.MiddlewareFunc{"createMatch": {router.Authentication(matchSession{})}}})
	req := httptest.NewRequest(http.MethodPost, "/api/match", strings.NewReader(`[{"programId":"owner"},{"programId":"owner"},{"programId":"owner"}]`))
	req.AddCookie(&http.Cookie{Name: "session", Value: "owner", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	if res.Code != 201 {
		t.Errorf("POST status=%d body=%s", res.Code, res.Body.String())
		return
	}
	var match api.Match
	if err := json.Unmarshal(res.Body.Bytes(), &match); err != nil {
		t.Errorf("decode match: %v", err)
		return
	}
	id, err := uuid.Parse(match.Id)
	if err != nil || id.Version() != 7 || match.RunId.Version() != 7 || match.RunId == id || match.Status == nil || *match.Status != api.Finished || match.OpponentName != "opponent" {
		t.Errorf("match=%+v", match)
	}
	run, err := NewRuns(pool).Get(t.Context(), match.RunId.String())
	if err != nil {
		t.Errorf("get persisted run: %v", err)
		return
	}
	if run.GoogleSubject != "owner" || run.OpponentSubject != "opponent" || run.FieldID != match.FieldId || run.TutorialID != "" || run.ProgramID != "" || !run.Result.IsValid() {
		t.Errorf("run=%+v", run)
	}
	var savedField string
	var before, after int
	if err := pool.QueryRow(t.Context(), "SELECT runs.field_id, matches.rating_before, matches.rating_after FROM matches JOIN runs ON runs.id=matches.run_id WHERE run_id=$1", match.RunId).Scan(&savedField, &before, &after); err != nil {
		t.Errorf("get persisted match: %v", err)
	}
	// The outcome depends on the engine, so only check that the saved result was applied
	// to the challenger alone; rating changes are covered by TestCreateMatchSavesRating.
	if before != domain.InitialRating || getRating(t, pool, "owner") != after || getRating(t, pool, "opponent") != domain.InitialRating {
		t.Errorf("rating=%d->%d, owner=%d, opponent=%d", before, after, getRating(t, pool, "owner"), getRating(t, pool, "opponent"))
	}
	if savedField != match.FieldId {
		t.Errorf("saved field=%s, response field=%s", savedField, match.FieldId)
	}
	if _, err := NewBattlePrograms(pool).Get(t.Context(), "owner"); !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("POST changed defense settings: %v", err)
	}
	for _, subject := range []string{"owner", "opponent", "stranger", ""} {
		t.Run("replay/"+subject, func(t *testing.T) {
			t.Parallel()
			for range 2 {
				fresh := echo.New()
				api.RegisterHandlersWithOptions(fresh, router.NewRouter(router.Config{RunResults: usecase.NewRunResults(NewRuns(pool))}), api.RegisterHandlersOptions{BaseURL: "/api", OperationMiddlewares: map[string][]echo.MiddlewareFunc{"runEvents": {router.Authentication(matchSession{})}}})
				req := httptest.NewRequest(http.MethodGet, "/api/run/"+match.RunId.String()+"/events", nil)
				if subject != "" {
					req.AddCookie(&http.Cookie{Name: "session", Value: subject, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
				}
				res := httptest.NewRecorder()
				fresh.ServeHTTP(res, req)
				want := 200
				switch subject {
				case "stranger":
					want = 403
				case "":
					want = 401
				}
				if res.Code != want {
					t.Errorf("GET status=%d want %d body=%s", res.Code, want, res.Body.String())
					continue
				}
				if want == 200 {
					compact, err := json.Marshal(run.Result)
					if err != nil {
						t.Errorf("compact result: %v", err)
						continue
					}
					if res.Body.String() != "event: result\ndata: "+string(compact)+"\n\n" {
						t.Errorf("replay differs from stored result: %s", res.Body.String())
					}
				}
			}
		})
	}
}

func ratedRecord(challenger, opponent string, outcome domain.MatchOutcome, delta int) domain.MatchRecord {
	return domain.MatchRecord{
		GoogleSubject: challenger, OpponentSubject: opponent, OpponentName: opponent, FieldID: "field",
		Result:  jsontext.Value(`{"history":[],"summary":{"score":{"player":0,"opponent":0}}}`),
		Outcome: outcome, ChallengerRating: domain.InitialRating, OpponentRating: domain.InitialRating, RatingDelta: delta,
	}
}

func getRating(t *testing.T, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, subject string) int {
	t.Helper()

	var rating int
	if err := pool.QueryRow(t.Context(), "SELECT rating FROM users WHERE google_subject=$1", subject).Scan(&rating); err != nil {
		t.Fatalf("get rating: %v", err)
	}
	return rating
}

func countRuns(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()

	var count int
	if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM runs").Scan(&count); err != nil {
		t.Fatalf("count runs: %v", err)
	}
	return count
}

func TestMatchSaveRollsBackRunAndRating(t *testing.T) {
	t.Parallel()

	pool := newTestPool(t)
	mustCreateUser(t, pool, "owner", ptr("Owner"))

	// The match row is inserted after the rating update, so its failure shows that
	// the update is rolled back together with the run.
	if _, err := NewMatches(pool).Create(t.Context(), ratedRecord("owner", "missing", domain.MatchWin, 30)); err == nil {
		t.Error("saving a nonexistent opponent succeeded")
	}

	if runs := countRuns(t, pool); runs != 0 {
		t.Errorf("failed match left %d runs", runs)
	}
	if got := getRating(t, pool, "owner"); got != domain.InitialRating {
		t.Errorf("failed match changed rating to %d", got)
	}
}

func TestMatchRating(t *testing.T) {
	t.Parallel()

	pool := newTestPool(t)
	mustCreateUser(t, pool, "owner", ptr("Owner"))
	matches := NewMatches(pool)

	if rating, err := matches.Rating(t.Context(), "owner"); err != nil || rating != domain.InitialRating {
		t.Errorf("Rating=%d, %v, want %d", rating, err, domain.InitialRating)
	}
	if _, err := matches.Rating(t.Context(), "missing"); !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("missing user error=%v, want %v", err, usecase.ErrNotFound)
	}
}

func TestCreateMatchSavesRating(t *testing.T) {
	t.Parallel()

	pool := newTestPool(t)
	mustCreateUser(t, pool, "owner", ptr("Owner"))
	mustCreateUser(t, pool, "opponent", ptr("Opponent"))

	match, err := NewMatches(pool).Create(t.Context(), ratedRecord("owner", "opponent", domain.MatchLoss, -25))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var outcome string
	var opponentRating, before, after int
	if err := pool.QueryRow(t.Context(), "SELECT outcome, opponent_rating, rating_before, rating_after FROM matches WHERE run_id=$1", match.RunID).
		Scan(&outcome, &opponentRating, &before, &after); err != nil {
		t.Fatalf("get match: %v", err)
	}
	if outcome != "loss" || opponentRating != 1000 || before != 1000 || after != 975 {
		t.Errorf("saved outcome=%s opponent=%d rating=%d->%d", outcome, opponentRating, before, after)
	}

	if got := getRating(t, pool, "owner"); got != 975 {
		t.Errorf("challenger rating=%d, want 975", got)
	}
	if got := getRating(t, pool, "opponent"); got != domain.InitialRating {
		t.Errorf("opponent rating changed to %d", got)
	}
}

// Concurrent results of one challenger, all rated from the same matching-time rating,
// are each added to the latest rating, and the saved histories chain without gaps.
func TestCreateMatchAppliesConcurrentResults(t *testing.T) {
	t.Parallel()

	pool := newTestPool(t)
	mustCreateUser(t, pool, "owner", ptr("Owner"))
	mustCreateUser(t, pool, "opponent", ptr("Opponent"))

	const results = 5
	var wg sync.WaitGroup
	for range results {
		wg.Go(func() {
			if _, err := NewMatches(pool).Create(t.Context(), ratedRecord("owner", "opponent", domain.MatchWin, 30)); err != nil {
				t.Errorf("Create: %v", err)
			}
		})
	}
	wg.Wait()

	if got, want := getRating(t, pool, "owner"), domain.InitialRating+results*30; got != want {
		t.Errorf("rating=%d, want %d", got, want)
	}

	rows, err := pool.Query(t.Context(), "SELECT rating_before, rating_after FROM matches ORDER BY rating_before")
	if err != nil {
		t.Fatal(err)
	}
	var history [][2]int
	for rows.Next() {
		var change [2]int
		if err := rows.Scan(&change[0], &change[1]); err != nil {
			t.Fatal(err)
		}
		history = append(history, change)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	var want [][2]int
	for i := range results {
		before := domain.InitialRating + i*30
		want = append(want, [2]int{before, before + 30})
	}
	if !reflect.DeepEqual(history, want) {
		t.Errorf("history=%v, want %v", history, want)
	}
}

func TestRanking(t *testing.T) {
	t.Parallel()

	pool := newTestPool(t)
	for _, subject := range []string{"alice", "bob", "carol", "dave", "idle"} {
		mustCreateUser(t, pool, subject, ptr(subject))
	}
	mustCreateUser(t, pool, "nameless", nil)
	mustCreateUser(t, pool, "blank", ptr(" "))

	matches := NewMatches(pool)
	matchesPlayed := []struct {
		challenger string
		outcome    domain.MatchOutcome
		delta      int
	}{
		{"alice", domain.MatchWin, 30}, {"alice", domain.MatchLoss, -25}, {"alice", domain.MatchDraw, 0},
		{"bob", domain.MatchWin, 30}, {"carol", domain.MatchWin, 30},
		{"dave", domain.MatchLoss, -25}, {"nameless", domain.MatchWin, 30}, {"blank", domain.MatchWin, 30},
	}
	for _, played := range matchesPlayed {
		if _, err := matches.Create(t.Context(), ratedRecord(played.challenger, "idle", played.outcome, played.delta)); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	// A match played before ratings were introduced still counts toward the record.
	if _, err := pool.Exec(t.Context(), `
		WITH run AS (
			INSERT INTO runs (id, google_subject, field_id, result)
			VALUES (gen_random_uuid(), 'dave', 'field', '{}') RETURNING id
		)
		INSERT INTO matches (id, run_id, opponent_google_subject, opponent_name, outcome)
		SELECT gen_random_uuid(), id, 'idle', 'idle', 'win' FROM run`); err != nil {
		t.Fatalf("insert legacy match: %v", err)
	}

	got, err := matches.Ranking(t.Context())
	if err != nil {
		t.Fatalf("Ranking: %v", err)
	}

	want := []domain.RankingEntry{
		{Rank: 1, Name: "bob", Rating: 1030, Wins: 1},
		{Rank: 1, Name: "carol", Rating: 1030, Wins: 1},
		{Rank: 3, Name: "alice", Rating: 1005, Wins: 1, Losses: 1, Draws: 1},
		{Rank: 4, Name: "dave", Rating: 975, Wins: 1, Losses: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Ranking=%+v, want %+v", got, want)
	}
}
