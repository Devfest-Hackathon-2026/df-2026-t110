package router

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

func (r *Router) GetMe(ctx *echo.Context) error {
	return errNotImplemented()
}

func (r *Router) SetMyUsername(ctx *echo.Context) error {
	if r.cfg.Sessions == nil || r.cfg.Users == nil {
		return errNotImplemented()
	}
	cookie, err := ctx.Request().Cookie("session")
	if err != nil || cookie.Value == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}
	user, err := r.cfg.Sessions.CurrentUser(ctx.Request().Context(), cookie.Value)
	if errors.Is(err, usecase.ErrUnauthenticated) {
		return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}
	if err != nil {
		return err
	}
	var request struct {
		Username string `json:"username"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(ctx.Response(), ctx.Request().Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}
	_, err = r.cfg.Users.SetUsername(ctx.Request().Context(), user.GoogleSubject, request.Username)
	switch {
	case errors.Is(err, usecase.ErrInvalidUsername):
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	case errors.Is(err, usecase.ErrUsernameAlreadySet), errors.Is(err, usecase.ErrUsernameTaken):
		return echo.NewHTTPError(http.StatusConflict, http.StatusText(http.StatusConflict))
	case errors.Is(err, usecase.ErrNotFound):
		return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	case err != nil:
		return err
	default:
		return ctx.NoContent(http.StatusNoContent)
	}
}
