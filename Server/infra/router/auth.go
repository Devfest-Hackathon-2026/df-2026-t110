package router

import (
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/labstack/echo/v5"
)

func (r *Router) Login(ctx *echo.Context) error {
	return errNotImplemented()
}

func (r *Router) LoginCallback(ctx *echo.Context, params api.LoginCallbackParams) error {
	return errNotImplemented()
}

func (r *Router) Logout(ctx *echo.Context) error {
	return errNotImplemented()
}
