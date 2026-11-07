package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/labstack/echo/v5"
)

type fixture struct {
	e *echo.Echo
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	r := NewRouter(Config{})
	e := echo.New()
	api.RegisterHandlers(e, r)
	return &fixture{e: e}
}

func TestRouterHandlersAreNotImplementedYet(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{name: "registered route is not implemented yet", method: http.MethodGet, path: "/tutorial", want: http.StatusNotImplemented},
		{name: "path parameter route is not implemented yet", method: http.MethodGet, path: "/field/field-1", want: http.StatusNotImplemented},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			response := httptest.NewRecorder()
			f.e.ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))
			if response.Code != tt.want {
				t.Errorf("%s %s status = %d, want %d", tt.method, tt.path, response.Code, tt.want)
			}
		})
	}
}

func TestRoomRoutesAreNotRegistered(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	for _, tt := range []struct{ method, path string }{
		{http.MethodGet, "/room"},
		{http.MethodGet, "/room/room-1"},
		{http.MethodGet, "/room/room-1/join"},
		{http.MethodPost, "/room/room-1/start/match-1"},
	} {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			f.e.ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))
			if response.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", response.Code, http.StatusNotFound)
			}
		})
	}
}
