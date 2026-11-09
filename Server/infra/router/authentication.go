package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

type SessionUserLookup interface {
	CurrentUser(ctx context.Context, sessionID string) (domain.User, error)
}

type authenticatedUserKey struct{}

// UserFromContext returns the user verified by Authentication for this request.
func UserFromContext(ctx context.Context) (domain.User, bool) {
	user, ok := ctx.Value(authenticatedUserKey{}).(domain.User)
	return user, ok
}

// Authentication verifies the session before calling the protected handler and
// puts the user in the request context using a private, typed key.
func Authentication(sessions SessionUserLookup) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx *echo.Context) error {
			if sessions == nil {
				return errNotImplemented()
			}
			cookie, err := ctx.Request().Cookie("session")
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)).Wrap(fmt.Errorf("read session cookie: %w", err))
			}
			if cookie.Value == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			}
			user, err := sessions.CurrentUser(ctx.Request().Context(), cookie.Value)
			if errors.Is(err, usecase.ErrUnauthenticated) {
				return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)).Wrap(fmt.Errorf("get current user: %w", err))
			}
			if err != nil {
				return fmt.Errorf("get current user: %w", err)
			}
			if user.GoogleSubject == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			}
			request := ctx.Request()
			ctx.SetRequest(request.WithContext(context.WithValue(request.Context(), authenticatedUserKey{}, user)))
			// Handler errors already carry their own context; middleware passes them through.
			return next(ctx)
		}
	}
}
