package router

import (
	json "encoding/json/v2"
	"fmt"
	"net/http"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/labstack/echo/v5"
)

func (r *Router) GetRanking(ctx *echo.Context) error {
	if r.cfg.Rankings == nil {
		return errNotImplemented()
	}
	entries, err := r.cfg.Rankings.List(ctx.Request().Context())
	if err != nil {
		return fmt.Errorf("list ranking: %w", err)
	}
	response := make([]api.RankingEntry, len(entries))
	for i, e := range entries {
		response[i] = api.RankingEntry{Rank: e.Rank, Name: e.Name, Rating: e.Rating, Wins: e.Wins, Losses: e.Losses, Draws: e.Draws}
	}
	body, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal ranking: %w", err)
	}
	return ctx.JSONBlob(http.StatusOK, body)
}
