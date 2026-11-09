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
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/tutorialrun"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type runSession struct{}

func (runSession) CurrentUser(context.Context, string) (domain.User, error) {
	return domain.User{GoogleSubject: "owner"}, nil
}

func TestTutorialRunPersistsResult(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		wait    bool
		cleared bool
		reason  string
		tick    int
	}{
		{name: "goal", cleared: true, reason: "condition_met", tick: 2},
		{name: "turn limit", wait: true, reason: "turn_limit", tick: 1000},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pool := newTestPool(t)
			mustCreateUser(t, pool, "owner", nil)
			programs := NewPrograms(pool)
			source, err := os.ReadFile("../tutorialrun/testdata/forward.json")
			if err != nil {
				t.Fatal(err)
			}
			if tt.wait {
				source = []byte(strings.Replace(strings.Replace(string(source), `"print_integer"`, `"wait"`, 1), "9223372036854775807", "1001", 1))
			}
			if err := programs.Create(t.Context(), domain.Program{ID: "p", GoogleSubject: "owner", Name: "forward", Source: source, TutorialID: ptr("tutorial-test")}); err != nil {
				t.Fatal(err)
			}
			tutorials, err := NewTextbooks(t.Context(), os.DirFS("testdata/tutorials"))
			if err != nil {
				t.Fatal(err)
			}
			runs := usecase.NewTutorialRuns(programs, tutorials, tutorialrun.NewExecutor(), NewRuns(pool))
			server := echo.New()
			api.RegisterHandlersWithOptions(server, router.NewRouter(router.Config{TutorialRuns: runs, RunResults: usecase.NewRunResults(NewRuns(pool))}), api.RegisterHandlersOptions{
				OperationMiddlewares: map[string][]echo.MiddlewareFunc{"runTutorial": {router.Authentication(runSession{})}, "runEvents": {router.Authentication(runSession{})}},
			})
			req := httptest.NewRequest(http.MethodPost, "/tutorial/tutorial-test/run", strings.NewReader(`{"programId":"p"}`))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: "session", Value: "s", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			created := httptest.NewRecorder()
			server.ServeHTTP(created, req)
			if created.Code != http.StatusCreated {
				t.Errorf("create status = %d: %s", created.Code, created.Body.String())
				return
			}
			var run api.TutorialRunCreated
			if err := json.Unmarshal(created.Body.Bytes(), &run); err != nil {
				t.Errorf("decode created run: %v", err)
				return
			}
			if run.RunId.Version() != 7 || run.FieldId != "field-test" {
				t.Errorf("created run = %+v", run)
			}
			// Read through a fresh repository to verify that POST saved the result in DB.
			stored, err := NewRuns(pool).Get(t.Context(), run.RunId.String())
			if err != nil {
				t.Errorf("read run: %v", err)
				return
			}
			if stored.GoogleSubject != "owner" || stored.TutorialID != "tutorial-test" || stored.ProgramID != "p" || stored.FieldID != run.FieldId {
				t.Errorf("run metadata = %+v", stored)
			}
			var result struct {
				History []jsontext.Value `json:"history"`
				Summary struct {
					Cleared bool   `json:"cleared"`
					Reason  string `json:"reason"`
					Tick    int    `json:"tick"`
				} `json:"summary"`
			}
			if err := json.Unmarshal(stored.Result, &result); err != nil {
				t.Errorf("decode saved result: %v", err)
				return
			}
			if result.Summary.Cleared != tt.cleared || result.Summary.Reason != tt.reason || result.Summary.Tick != tt.tick || len(result.History) == 0 {
				t.Errorf("saved result = %s", stored.Result)
			}
			if !tt.wait && !strings.Contains(string(stored.Result), "9223372036854775807") {
				t.Error("saved history lost the large integer")
			}
			eventsReq := httptest.NewRequest(http.MethodGet, "/run/"+run.RunId.String()+"/events", nil)
			eventsReq.AddCookie(&http.Cookie{Name: "session", Value: "s", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			events := httptest.NewRecorder()
			server.ServeHTTP(events, eventsReq)
			if events.Code != http.StatusOK {
				t.Errorf("events status = %d: %s", events.Code, events.Body.String())
			}
			data := strings.TrimSuffix(strings.TrimPrefix(events.Body.String(), "event: result\ndata: "), "\n\n")
			var retrieved any
			var saved any
			if err := json.Unmarshal([]byte(data), &retrieved); err != nil {
				t.Errorf("decode retrieved result: %v", err)
			}
			if err := json.Unmarshal(stored.Result, &saved); err != nil {
				t.Errorf("decode saved result: %v", err)
			}
			if !reflect.DeepEqual(retrieved, saved) {
				t.Error("retrieved result differs from saved history and summary")
			}
			if _, err := NewRuns(pool).Get(t.Context(), uuid.NewString()); !errors.Is(err, usecase.ErrNotFound) {
				t.Errorf("unknown run: %v", err)
			}
		})
	}
}
