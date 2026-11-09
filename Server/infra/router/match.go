package router

import (
	json "encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

func (r *Router) CreateMatch(ctx *echo.Context) error {
	if r.cfg.Matches == nil {
		return errNotImplemented()
	}
	user, ok := UserFromContext(ctx.Request().Context())
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}
	var request api.CreateMatchJSONRequestBody
	if err := json.UnmarshalRead(ctx.Request().Body, &request, json.RejectUnknownMembers(true)); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest)).Wrap(err)
	}
	if len(request) != domain.BattleRobotCount {
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}
	ids := make([]string, len(request))
	for i, slot := range request {
		if strings.TrimSpace(slot.ProgramId) == "" {
			return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		}
		ids[i] = slot.ProgramId
	}
	match, err := r.cfg.Matches.Create(ctx.Request().Context(), user.GoogleSubject, ids)
	if errors.Is(err, usecase.ErrNoOpponent) {
		return echo.NewHTTPError(http.StatusConflict, http.StatusText(http.StatusConflict)).Wrap(err)
	}
	if err != nil {
		return battleProgramError(ctx, err)
	}
	body, err := json.Marshal(struct {
		ID               string `json:"id"`
		FieldID          string `json:"fieldId"`
		SelectedLayoutID string `json:"selectedLayoutId"`
		OpponentName     string `json:"opponentName"`
		RunID            string `json:"runId"`
		Status           string `json:"status"`
	}{match.ID, match.FieldID, match.SelectedLayoutID, match.OpponentName, match.RunID, "finished"})
	if err != nil {
		return fmt.Errorf("marshal match: %w", err)
	}
	return ctx.JSONBlob(http.StatusCreated, body)
}

func (r *Router) GetBattlePrograms(ctx *echo.Context) error {
	if r.cfg.BattlePrograms == nil {
		return errNotImplemented()
	}
	user, ok := UserFromContext(ctx.Request().Context())
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}
	ids, err := r.cfg.BattlePrograms.Get(ctx.Request().Context(), user.GoogleSubject)
	if err != nil {
		return battleProgramError(ctx, err)
	}
	body, err := json.Marshal(ids)
	if err != nil {
		return fmt.Errorf("marshal battle programs: %w", err)
	}
	return ctx.JSONBlob(http.StatusOK, body)
}

func (r *Router) SetBattlePrograms(ctx *echo.Context) error {
	if r.cfg.BattlePrograms == nil {
		return errNotImplemented()
	}
	user, ok := UserFromContext(ctx.Request().Context())
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}
	var ids api.SetBattleProgramsJSONRequestBody
	if err := json.UnmarshalRead(ctx.Request().Body, &ids); err != nil {
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
