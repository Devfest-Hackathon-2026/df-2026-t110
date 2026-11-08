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
)

type sessionLookupStub struct{ err error }

func (s sessionLookupStub) CurrentUser(_ context.Context, _ string) (domain.User, error) {
	return domain.User{GoogleSubject: "verified-subject"}, s.err
}

type usernameSetterStub struct {
	subject, name string
	err           error
}

func (s *usernameSetterStub) SetUsername(_ context.Context, subject, name string) (domain.User, error) {
	s.subject, s.name = subject, name
	return domain.User{}, s.err
}

func TestSetMyUsername(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, body         string
		cookie             bool
		sessionErr, setErr error
		want               int
	}{
		{"success", `{"username":"Alice"}`, true, nil, nil, http.StatusNoContent},
		{"missing cookie", `{"username":"Alice"}`, false, nil, nil, http.StatusUnauthorized},
		{"expired session", `{"username":"Alice"}`, true, usecase.ErrUnauthenticated, nil, http.StatusUnauthorized},
		{"invalid name", `{"username":"!"}`, true, nil, usecase.ErrInvalidUsername, http.StatusBadRequest},
		{"already set", `{"username":"Alice"}`, true, nil, usecase.ErrUsernameAlreadySet, http.StatusConflict},
		{"taken", `{"username":"Alice"}`, true, nil, usecase.ErrUsernameTaken, http.StatusConflict},
		{"untrusted subject", `{"username":"Alice","subject":"other"}`, true, nil, nil, http.StatusBadRequest},
		{"multiple bodies", `{"username":"Alice"}{}`, true, nil, nil, http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			setter := &usernameSetterStub{err: tt.setErr}
			r := NewRouter(Config{Users: setter})
			e := echo.New()
			api.RegisterHandlersWithOptions(e, r, api.RegisterHandlersOptions{
				OperationMiddlewares: map[string][]echo.MiddlewareFunc{
					"setMyUsername": {Authentication(sessionLookupStub{err: tt.sessionErr})},
				},
			})
			req := httptest.NewRequest(http.MethodPost, "/me/username", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.cookie {
				req.AddCookie(&http.Cookie{Name: "session", Value: "session-id", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			}
			res := httptest.NewRecorder()
			e.ServeHTTP(res, req)
			if res.Code != tt.want {
				t.Errorf("status = %d, want %d", res.Code, tt.want)
			}
			if tt.want == http.StatusNoContent && (setter.subject != "verified-subject" || setter.name != "Alice") {
				t.Errorf("setter received %q, %q", setter.subject, setter.name)
			}
			if tt.want == http.StatusUnauthorized && setter.subject != "" {
				t.Error("setter called without authentication")
			}
		})
	}
}
