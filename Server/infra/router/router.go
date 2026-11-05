package router

import (
	"net/http"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/labstack/echo/v5"
)

var _ api.ServerInterface = (*Router)(nil)

type Router struct{}

func NewRouter() (*Router, error) {
	return &Router{}, nil
}

func errNotImplemented() error {
	return echo.NewHTTPError(http.StatusNotImplemented, http.StatusText(http.StatusNotImplemented))
}
