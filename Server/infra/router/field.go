package router

import (
	json "encoding/json/v2"
	"errors"
	"fmt"
	"net/http"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

func (r *Router) GetField(ctx *echo.Context, fieldID api.FieldId) error {
	if r.cfg.Fields == nil {
		return errNotImplemented()
	}
	field, err := r.cfg.Fields.Get(ctx.Request().Context(), fieldID)
	if errors.Is(err, usecase.ErrNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	}
	if err != nil {
		return fmt.Errorf("get field %q: %w", fieldID, err)
	}
	data, err := json.Marshal(field)
	if err != nil {
		return fmt.Errorf("marshal field %q: %w", fieldID, err)
	}
	return ctx.Blob(http.StatusOK, "application/json", data)
}
