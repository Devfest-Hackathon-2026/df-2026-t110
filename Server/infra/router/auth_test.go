package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

type authStub struct{ callback usecase.LoginCallback }

func (a *authStub) BeginLogin(context.Context) (usecase.LoginAttempt, error) {
	return usecase.LoginAttempt{ID: "attempt-1", Request: usecase.AuthorizationRequest{URL: "https://accounts.example/authorize"}, ExpiresAt: time.Now().Add(time.Minute)}, nil
}
func (a *authStub) CompleteLogin(_ context.Context, callback usecase.LoginCallback) (domain.Session, error) {
	a.callback = callback
	if callback.State != "state" || callback.Code != "code" {
		return domain.Session{}, usecase.ErrInvalidLogin
	}
	return domain.Session{ID: "session-1", ExpiresAt: time.Now().Add(time.Hour)}, nil
}

type authSessionStub struct{ deleted string }

func (s *authSessionStub) CurrentUser(context.Context, string) (domain.User, error) {
	name := "Ada"
	return domain.User{GoogleSubject: "google-subject", Name: &name}, nil
}
func (s *authSessionStub) Logout(_ context.Context, id string) error { s.deleted = id; return nil }

func TestAuthRoutes(t *testing.T) {
	t.Parallel()
	auth, sessions := &authStub{}, &authSessionStub{}
	e := echo.New()
	api.RegisterHandlersWithOptions(e, NewRouter(Config{Auth: auth, Sessions: sessions, SecureCookies: true}), api.RegisterHandlersOptions{
		BaseURL: "/api", OperationMiddlewares: map[string][]echo.MiddlewareFunc{
			"getMe": {Authentication(sessions)}, "logout": {Authentication(sessions)},
		},
	})
	call := func(method, path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, nil)
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		res := httptest.NewRecorder()
		e.ServeHTTP(res, req)
		return res
	}
	login := call(http.MethodGet, "/api/login")
	if login.Code != http.StatusFound || login.Header().Get("Location") != "https://accounts.example/authorize" {
		t.Fatalf("login = %d %s", login.Code, login.Header().Get("Location"))
	}
	if cookie := login.Result().Cookies()[0]; cookie.Name != "login_attempt" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("login cookie = %+v", cookie)
	}
	if got := call(http.MethodGet, "/api/login/callback?state=state&code=code"); got.Code != http.StatusBadRequest {
		t.Errorf("callback without attempt = %d", got.Code)
	}
	callback := call(http.MethodGet, "/api/login/callback?state=state&code=code", &http.Cookie{Name: "login_attempt", Value: "attempt-1"})
	if callback.Code != http.StatusFound || callback.Header().Get("Location") != "/account" {
		t.Fatalf("callback = %d %s", callback.Code, callback.Header().Get("Location"))
	}
	if auth.callback.AttemptID != "attempt-1" {
		t.Errorf("attempt ID = %q", auth.callback.AttemptID)
	}
	var sessionCookie *http.Cookie
	for _, cookie := range callback.Result().Cookies() {
		if cookie.Name == "session" {
			sessionCookie = cookie
		}
	}
	if sessionCookie == nil || sessionCookie.Value != "session-1" || !sessionCookie.HttpOnly || !sessionCookie.Secure {
		t.Fatalf("session cookie = %+v", sessionCookie)
	}
	if got := call(http.MethodGet, "/api/me"); got.Code != http.StatusUnauthorized {
		t.Errorf("anonymous me = %d", got.Code)
	}
	me := call(http.MethodGet, "/api/me", sessionCookie)
	if me.Code != http.StatusOK || strings.TrimSpace(me.Body.String()) != `{"id":"google-subject","name":"Ada"}` {
		t.Errorf("me = %d %s", me.Code, me.Body.String())
	}
	logout := call(http.MethodPost, "/api/logout", sessionCookie)
	if logout.Code != http.StatusNoContent || sessions.deleted != "session-1" {
		t.Errorf("logout = %d, deleted %q", logout.Code, sessions.deleted)
	}
	if cookie := logout.Result().Cookies()[0]; cookie.Name != "session" || cookie.MaxAge >= 0 {
		t.Errorf("logout cookie = %+v", cookie)
	}
}
