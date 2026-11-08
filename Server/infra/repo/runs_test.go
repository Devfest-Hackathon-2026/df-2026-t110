package repo

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
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
	pool := newTestPool(t)
	mustCreateUser(t, pool, "owner", nil)
	programs := NewPrograms(pool)
	source, err := os.ReadFile("../tutorialrun/testdata/forward.json")
	if err != nil {
		t.Fatal(err)
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
	api.RegisterHandlersWithOptions(server, router.NewRouter(router.Config{TutorialRuns: runs}), api.RegisterHandlersOptions{
		OperationMiddlewares: map[string][]echo.MiddlewareFunc{"runTutorial": {router.Authentication(runSession{})}},
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
			Cleared bool `json:"cleared"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(stored.Result, &result); err != nil {
		t.Errorf("decode saved result: %v", err)
		return
	}
	if !result.Summary.Cleared || len(result.History) == 0 || !strings.Contains(string(stored.Result), "9223372036854775807") {
		t.Errorf("saved result = %s", stored.Result)
	}
	if _, err := NewRuns(pool).Get(t.Context(), uuid.NewString()); !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("unknown run: %v", err)
	}
}
