package router

import (
	"errors"
	"net/http"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

func (r *Router) Login(ctx *echo.Context) error {
	if r.cfg.Auth == nil {
		return errNotImplemented()
	}
	attempt, err := r.cfg.Auth.BeginLogin(ctx.Request().Context())
	if errors.Is(err, usecase.ErrLoginUnavailable) {
		return errNotImplemented()
	}
	if err != nil {
		return err
	}
	ctx.SetCookie(&http.Cookie{Name: "login_attempt", Value: attempt.ID, Path: "/api/login", HttpOnly: true, Secure: r.cfg.SecureCookies, SameSite: http.SameSiteLaxMode, Expires: attempt.ExpiresAt})
	return ctx.Redirect(http.StatusFound, attempt.Request.URL)
}

func (r *Router) LoginCallback(ctx *echo.Context, params api.LoginCallbackParams) error {
	if r.cfg.Auth == nil {
		return errNotImplemented()
	}
	cookie, err := ctx.Request().Cookie("login_attempt")
	if err != nil || cookie.Value == "" {
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}
	ctx.SetCookie(&http.Cookie{Name: "login_attempt", Path: "/api/login", HttpOnly: true, Secure: r.cfg.SecureCookies, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	callback := usecase.LoginCallback{AttemptID: cookie.Value, State: params.State}
	if params.Code != nil {
		callback.Code = *params.Code
	}
	if params.Error != nil {
		callback.ProviderError = *params.Error
	}
	session, err := r.cfg.Auth.CompleteLogin(ctx.Request().Context(), callback)
	if errors.Is(err, usecase.ErrInvalidLogin) {
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest)).Wrap(err)
	}
	if errors.Is(err, usecase.ErrLoginUnavailable) {
		return errNotImplemented()
	}
	if err != nil {
		return err
	}
	ctx.SetCookie(&http.Cookie{Name: "session", Value: session.ID, Path: "/", HttpOnly: true, Secure: r.cfg.SecureCookies, SameSite: http.SameSiteLaxMode, Expires: session.ExpiresAt})
	return ctx.Redirect(http.StatusFound, "/account")
}

func (r *Router) Logout(ctx *echo.Context) error {
	if r.cfg.Sessions == nil {
		return errNotImplemented()
	}
	cookie, err := ctx.Request().Cookie("session")
	if err != nil || cookie.Value == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}
	if err := r.cfg.Sessions.Logout(ctx.Request().Context(), cookie.Value); err != nil {
		return err
	}
	ctx.SetCookie(&http.Cookie{Name: "session", Path: "/", HttpOnly: true, Secure: r.cfg.SecureCookies, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(0, 0)})
	return ctx.NoContent(http.StatusNoContent)
}
