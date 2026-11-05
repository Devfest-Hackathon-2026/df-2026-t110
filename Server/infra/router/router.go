package router

import (
	"net/http"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

var _ api.ServerInterface = (*Router)(nil)

// DefaultHeartbeatInterval keeps idle event streams alive through proxies that
// close connections after tens of seconds of silence.
const DefaultHeartbeatInterval = 15 * time.Second

// Config holds HTTP-level settings of the handlers.
type Config struct {
	// HeartbeatInterval defaults to DefaultHeartbeatInterval when not positive.
	HeartbeatInterval time.Duration
}

// Router translates HTTP requests into use case calls.
type Router struct {
	roomEvents *usecase.RoomEvents
	cfg        Config
}

func NewRouter(roomEvents *usecase.RoomEvents, cfg Config) *Router {
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = DefaultHeartbeatInterval
	}
	return &Router{roomEvents: roomEvents, cfg: cfg}
}

func errNotImplemented() error {
	return echo.NewHTTPError(http.StatusNotImplemented, http.StatusText(http.StatusNotImplemented))
}
