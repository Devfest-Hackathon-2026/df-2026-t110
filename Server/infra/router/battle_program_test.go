package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

type battleProgramStore struct{ saved []string }

func (s *battleProgramStore) Set(_ context.Context, _ string, ids []string) error {
	s.saved = ids
	return nil
}

type battleProgramLookup struct{}

func (battleProgramLookup) Get(_ context.Context, id string) (domain.Program, error) {
	return domain.Program{ID: id, GoogleSubject: "verified-subject", IsBattle: true}, nil
}
func TestPutBattlePrograms(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		body   string
		cookie bool
		want   int
	}{
		{`["a","a","a"]`, true, 204}, {`["a","b"]`, true, 400}, {`["a","b","c","d"]`, true, 400},
		{`null`, true, 400}, {`[null,"a","b"]`, true, 400}, {`["a","b","c"] {}`, true, 400}, {`{}`, true, 400}, {`["a","a","a"]`, false, 401},
	} {
		t.Run(tt.body+http.StatusText(tt.want), func(t *testing.T) {
			t.Parallel()
			store := &battleProgramStore{}
			e := echo.New()
			r := NewRouter(Config{BattlePrograms: usecase.NewBattlePrograms(store, battleProgramLookup{})})
			api.RegisterHandlersWithOptions(e, r, api.RegisterHandlersOptions{OperationMiddlewares: map[string][]echo.MiddlewareFunc{"setBattlePrograms": {Authentication(sessionLookupStub{}), middleware.BodyLimit(1 << 20)}}})
			req := httptest.NewRequest(http.MethodPut, "/battle/program", strings.NewReader(tt.body))
			if tt.cookie {
				req.AddCookie(&http.Cookie{Name: "session", Value: "session-id", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			}
			res := httptest.NewRecorder()
			e.ServeHTTP(res, req)
			if res.Code != tt.want {
				t.Errorf("status=%d body=%s, want %d", res.Code, res.Body.String(), tt.want)
			}
			if tt.want != 204 && store.saved != nil {
				t.Error("invalid request saved")
			}
		})
	}
}

func (s *battleProgramStore) Get(_ context.Context, _ string) ([]string, error) { return s.saved, nil }

func TestGetBattlePrograms(t *testing.T) {
	t.Parallel()
	for _, cookie := range []bool{true, false} {
		t.Run(http.StatusText(map[bool]int{true: 200, false: 401}[cookie]), func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			store := &battleProgramStore{saved: []string{"c", "a", "c"}}
			r := NewRouter(Config{BattlePrograms: usecase.NewBattlePrograms(store, battleProgramLookup{})})
			api.RegisterHandlersWithOptions(e, r, api.RegisterHandlersOptions{OperationMiddlewares: map[string][]echo.MiddlewareFunc{"getBattlePrograms": {Authentication(sessionLookupStub{})}}})
			req := httptest.NewRequest(http.MethodGet, "/battle/program", nil)
			if cookie {
				req.AddCookie(&http.Cookie{Name: "session", Value: "session-id", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			}
			res := httptest.NewRecorder()
			e.ServeHTTP(res, req)
			want := 401
			if cookie {
				want = 200
			}
			if res.Code != want {
				t.Errorf("status=%d, want %d", res.Code, want)
			}
			if cookie && strings.TrimSpace(res.Body.String()) != `["c","a","c"]` {
				t.Errorf("body=%s", res.Body.String())
			}
		})
	}
}
