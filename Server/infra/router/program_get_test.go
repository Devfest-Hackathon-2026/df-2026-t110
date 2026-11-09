package router

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

func TestGetProgram(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, owner                              string
		cookie, tutorial, noPrograms, noSessions bool
		withoutMiddleware                        bool
		getErr, sessionErr                       error
		want                                     int
	}{
		{name: "owner battle draft", cookie: true, want: 200},
		{name: "owner tutorial draft", cookie: true, tutorial: true, want: 200},
		{name: "missing", cookie: true, getErr: usecase.ErrNotFound, want: 404},
		{name: "other user", cookie: true, owner: "other-subject", want: 403},
		{name: "no session", want: 401},
		{name: "expired session", cookie: true, sessionErr: usecase.ErrUnauthenticated, want: 401},
		{name: "handler without authentication middleware", withoutMiddleware: true, want: 401},
		{name: "session failure", cookie: true, sessionErr: errors.New("session storage unavailable"), want: 500},
		{name: "repository failure", cookie: true, getErr: errors.New("program storage unavailable"), want: 500},
		{name: "programs not configured", cookie: true, noPrograms: true, want: 501},
		{name: "sessions not configured", cookie: true, noSessions: true, want: 501},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			owner := tt.owner
			if owner == "" {
				owner = "verified-subject"
			}
			want := domain.Program{ID: "path-id", GoogleSubject: owner, Name: "saved draft", IsBattle: !tt.tutorial, Source: jsontext.Value(programSource)}
			if tt.tutorial {
				tutorialID := "tutorial-1"
				want.TutorialID = &tutorialID
			}
			store := &programStore{program: want, getErr: tt.getErr}
			cfg := Config{}
			if !tt.noPrograms {
				cfg.Programs = usecase.NewPrograms(store, ProgramSourceValidator{})
			}
			var sessions SessionUserLookup = sessionLookupStub{err: tt.sessionErr}
			if tt.noSessions {
				sessions = nil
			}
			options := api.RegisterHandlersOptions{BaseURL: "/api"}
			if !tt.withoutMiddleware {
				options.OperationMiddlewares = map[string][]echo.MiddlewareFunc{"getProgram": {Authentication(sessions)}}
			}
			e := echo.New()
			api.RegisterHandlersWithOptions(e, NewRouter(cfg), options)
			req := httptest.NewRequest(http.MethodGet, "/api/program/path-id", nil)
			if tt.cookie {
				req.AddCookie(&http.Cookie{Name: "session", Value: "session-id", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			}
			res := httptest.NewRecorder()
			e.ServeHTTP(res, req)
			if res.Code != tt.want {
				t.Errorf("status = %d, want %d; body = %s", res.Code, tt.want, res.Body.String())
			}
			if len(store.saved) != 0 || len(store.updated) != 0 {
				t.Error("GET wrote program data")
			}
			if tt.want != http.StatusOK {
				var publicError map[string]string
				if err := json.Unmarshal(res.Body.Bytes(), &publicError); err != nil {
					t.Errorf("decode error response: %v", err)
				} else if len(publicError) != 1 || publicError["message"] != http.StatusText(tt.want) {
					t.Errorf("unexpected public error response: %s", res.Body.String())
				}
				if (tt.want == http.StatusUnauthorized || tt.want == http.StatusNotImplemented) && store.getID != "" {
					t.Error("unconfigured or unauthenticated request reached the repository")
				}
				return
			}
			if store.getID != "path-id" {
				t.Errorf("repository ID = %q, want path-id", store.getID)
			}
			if res.Header().Get("Cache-Control") != "no-store" {
				t.Errorf("Cache-Control = %q, want no-store", res.Header().Get("Cache-Control"))
			}
			var got struct {
				ID         string         `json:"id"`
				Name       string         `json:"name"`
				Source     jsontext.Value `json:"source"`
				IsBattle   bool           `json:"isBattle"`
				TutorialID *string        `json:"tutorialID,omitempty"`
			}
			if err := json.Unmarshal(res.Body.Bytes(), &got, json.RejectUnknownMembers(true)); err != nil {
				t.Errorf("decode SavedProgram: %v", err)
				return
			}
			if got.ID != want.ID || got.Name != want.Name || string(got.Source) != programSource || got.IsBattle != want.IsBattle || !reflect.DeepEqual(got.TutorialID, want.TutorialID) {
				t.Errorf("SavedProgram = %+v, want %+v", got, want)
			}
		})
	}
}
