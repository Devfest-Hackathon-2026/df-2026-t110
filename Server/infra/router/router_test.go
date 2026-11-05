package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/labstack/echo/v5"
)

func TestRouterHandlersAreNotImplementedYet(t *testing.T) {
	t.Parallel()

	router, err := NewRouter()
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}
	e := echo.New()
	api.RegisterHandlers(e, router)

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
			e.ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))
			if response.Code != tt.want {
				t.Errorf("%s %s status = %d, want %d", tt.method, tt.path, response.Code, tt.want)
			}
		})
	}
}
