package repo

import (
	"encoding/json/jsontext"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

func TestRunEventsReplaysSavedResult(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	mustCreateUser(t, pool, "owner", nil)
	if err := NewPrograms(pool).Create(t.Context(), domain.Program{ID: "p", GoogleSubject: "owner", Name: "test", Source: jsontext.Value(`{}`), TutorialID: ptr("tutorial-test")}); err != nil {
		t.Fatal(err)
	}
	runs := NewRuns(pool)
	id, err := runs.Create(t.Context(), domain.Run{GoogleSubject: "owner", TutorialID: "tutorial-test", FieldID: "field-test", ProgramID: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if err := runs.SaveResult(t.Context(), id, jsontext.Value(`{"history":[],"summary":{"cleared":true,"tick":1,"reason":"condition_met"}}`)); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		// Replay through a fresh handler and repository, with no executor configured.
		server := echo.New()
		api.RegisterHandlersWithOptions(server, router.NewRouter(router.Config{RunResults: usecase.NewRunResults(NewRuns(pool))}), api.RegisterHandlersOptions{
			OperationMiddlewares: map[string][]echo.MiddlewareFunc{"runEvents": {router.Authentication(runSession{})}},
		})
		req := httptest.NewRequest(http.MethodGet, "/run/"+id+"/events", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: "s", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
		res := httptest.NewRecorder()
		server.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Errorf("replay status = %d: %s", res.Code, res.Body.String())
			continue
		}
		if !strings.HasPrefix(res.Body.String(), "event: result\ndata: ") || !strings.Contains(res.Body.String(), `"cleared":true`) || !strings.Contains(res.Body.String(), `"reason":"condition_met"`) {
			t.Errorf("replayed result = %s", res.Body.String())
		}
	}
}
