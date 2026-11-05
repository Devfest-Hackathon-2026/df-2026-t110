package router

import (
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/labstack/echo/v5"
)

func (r *Router) GetField(ctx *echo.Context, fieldID api.FieldId) error {
	return errNotImplemented()
}
