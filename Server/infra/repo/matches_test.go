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
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/matchrun"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

func TestDrawBattleOpponent(t *testing.T) {
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
		team, err := NewMatches(pool).Draw(t.Context(), "owner")
		if err != nil {
			t.Errorf("Draw: %v", err)
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
	if _, err := NewMatches(pool).Draw(t.Context(), "owner"); !errors.Is(err, usecase.ErrNoOpponent) {
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
	source, err := os.ReadFile("../tutorialrun/testdata/forward.json")
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
	server := echo.New()
	api.RegisterHandlersWithOptions(server, router.NewRouter(router.Config{Matches: usecase.NewMatches(usecase.MatchDeps{
		Users: NewUsers(pool), Programs: programs, Opponents: matches, Layouts: matchrun.NewLayouts(), Executor: matchrun.NewExecutor(), Matches: matches,
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
	var savedLayout string
	if err := pool.QueryRow(t.Context(), "SELECT selected_layout_id FROM matches WHERE run_id=$1", match.RunId).Scan(&savedLayout); err != nil {
		t.Errorf("get persisted match: %v", err)
	}
	if savedLayout != match.SelectedLayoutId {
		t.Errorf("saved layout=%s, response layout=%s", savedLayout, match.SelectedLayoutId)
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

func TestMatchSaveRollsBackRun(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	mustCreateUser(t, pool, "owner", ptr("Owner"))
	_, err := NewMatches(pool).Create(t.Context(), domain.MatchRecord{
		GoogleSubject: "owner", OpponentSubject: "missing", OpponentName: "Opponent", FieldID: "field", SelectedLayoutID: "layout", Result: jsontext.Value(`{"history":[],"summary":{"score":{"player":0,"opponent":0}}}`),
	})
	if err == nil {
		t.Error("saving a nonexistent opponent succeeded")
	}
	var count int
	if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM runs").Scan(&count); err != nil {
		t.Errorf("count runs: %v", err)
		return
	}
	if count != 0 {
		t.Errorf("failed match left %d runs", count)
	}
}
