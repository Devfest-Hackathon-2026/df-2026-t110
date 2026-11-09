package router

import (
	json "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

func (r *Router) CreateMatch(ctx *echo.Context) error {
	return errNotImplemented()
}

func (r *Router) GetBattlePrograms(ctx *echo.Context) error {
	return errNotImplemented()
}

func (r *Router) SetBattlePrograms(ctx *echo.Context) error {
	if r.cfg.BattlePrograms == nil {
		return errNotImplemented()
	}
	user, ok := UserFromContext(ctx.Request().Context())
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}
	body, err := io.ReadAll(ctx.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest)).Wrap(err)
	}
	var ids []string
	if err := json.Unmarshal(body, &ids); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest)).Wrap(err)
	}
	return battleProgramError(ctx, r.cfg.BattlePrograms.Set(ctx.Request().Context(), user.GoogleSubject, ids))
}

func battleProgramError(ctx *echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidProgram):
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest)).Wrap(err)
	case errors.Is(err, usecase.ErrUnauthenticated):
		return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)).Wrap(err)
	case errors.Is(err, usecase.ErrForbidden):
		return echo.NewHTTPError(http.StatusForbidden, http.StatusText(http.StatusForbidden)).Wrap(err)
	case errors.Is(err, usecase.ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, http.StatusText(http.StatusNotFound)).Wrap(err)
	case err != nil:
		return fmt.Errorf("battle programs: %w", err)
	}
	return ctx.NoContent(http.StatusNoContent)
}
