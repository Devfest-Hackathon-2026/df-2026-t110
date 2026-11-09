package repo

import (
	"encoding/json/jsontext"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

func TestInvalidUpdatePreservesProgramAndRevision(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	mustCreateUser(t, pool, "owner", nil)
	source, err := os.ReadFile("../tutorialrun/testdata/forward.json")
	if err != nil {
		t.Fatal(err)
	}
	compact := jsontext.Value(source)
	if err := compact.Compact(); err != nil {
		t.Fatal(err)
	}
	programs := NewPrograms(pool)
	if err := programs.Create(t.Context(), domain.Program{ID: "saved", GoogleSubject: "owner", Name: "original", Source: compact, IsBattle: true}); err != nil {
		t.Fatal(err)
	}
	original, err := programs.Get(t.Context(), "saved")
	if err != nil {
		t.Fatal(err)
	}
	if err := original.Source.Compact(); err != nil {
		t.Fatal(err)
	}
	server := echo.New()
	api.RegisterHandlersWithOptions(server, router.NewRouter(router.Config{Programs: usecase.NewPrograms(programs, router.ProgramSourceValidator{})}), api.RegisterHandlersOptions{OperationMiddlewares: map[string][]echo.MiddlewareFunc{"updateProgram": {router.Authentication(runSession{})}}})
	for _, invalid := range []string{
		strings.Replace(string(compact), `"func":"print_integer"`, `"func":"print_bool"`, 1),
		strings.Replace(string(compact), `"func":"print_integer"`, `"func":"missing"`, 1),
	} {
		req := httptest.NewRequest(http.MethodPut, "/program/saved", strings.NewReader(`{"name":"changed","source":`+invalid+`}`))
		req.AddCookie(&http.Cookie{Name: "session", Value: "owner", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
		res := httptest.NewRecorder()
		server.ServeHTTP(res, req)
		if res.Code != 400 {
			t.Errorf("status=%d, body=%s", res.Code, res.Body.String())
		}
	}
	got, err := programs.Get(t.Context(), "saved")
	if err != nil {
		t.Errorf("read unchanged program: %v", err)
	} else {
		if err := got.Source.Compact(); err != nil {
			t.Errorf("compact stored source: %v", err)
		}
		if got.Name != "original" || got.Revision != 1 || string(got.Source) != string(original.Source) {
			t.Errorf("invalid update changed program: %+v", got)
		}
	}
	var count int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM program_revisions WHERE program_id='saved'`).Scan(&count); err != nil {
		t.Errorf("read revisions: %v", err)
	} else if count != 1 {
		t.Errorf("invalid updates created revisions: %d", count)
	}
}
