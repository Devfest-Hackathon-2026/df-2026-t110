package router

import (
	"context"
	"net/http"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/labstack/echo/v5"
)

var _ api.ServerInterface = (*Router)(nil)

// Config holds HTTP-level settings of the handlers.
type Config struct {
	Sessions SessionUserLookup
	Users    UsernameSetter
}

type SessionUserLookup interface {
	CurrentUser(ctx context.Context, sessionID string) (domain.User, error)
}

type UsernameSetter interface {
	SetUsername(ctx context.Context, subject, name string) (domain.User, error)
}

// Router translates HTTP requests into use case calls.
type Router struct {
	cfg Config
}

func NewRouter(cfg Config) *Router {
	return &Router{cfg: cfg}
}

func errNotImplemented() error {
	return echo.NewHTTPError(http.StatusNotImplemented, http.StatusText(http.StatusNotImplemented))
}
