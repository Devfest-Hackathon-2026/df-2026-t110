package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/labstack/echo/v5"
)

func TestProgramGenerationRouteRemoved(t *testing.T) {
	t.Parallel()

	for _, authenticated := range []bool{false, true} {
		name := "without session"
		if authenticated {
			name = "with session"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			api.RegisterHandlersWithBaseURL(e, NewRouter(Config{}), "/api")
			req := httptest.NewRequest(http.MethodPost, "/api/program/create-with-vibe", strings.NewReader(`{"prompt":"go forward"}`))
			req.Header.Set("Content-Type", "application/json")
			if authenticated {
				req.AddCookie(&http.Cookie{Name: "session", Value: "session-id", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			}
			res := httptest.NewRecorder()
			e.ServeHTTP(res, req)

			// The remaining /program/:programId route only accepts GET and PUT.
			if res.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d; body = %s", res.Code, http.StatusMethodNotAllowed, res.Body.String())
			}
		})
	}
}
