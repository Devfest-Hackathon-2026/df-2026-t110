package router

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"net/http"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

type tutorialSummary struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func (r *Router) ListTutorials(ctx *echo.Context) error {
	if r.cfg.Tutorials == nil {
		return errNotImplemented()
	}
	tutorials, err := r.cfg.Tutorials.List(ctx.Request().Context())
	if err != nil {
		return fmt.Errorf("list tutorials: %w", err)
	}
	list := make([]tutorialSummary, 0, len(tutorials))
	for _, tutorial := range tutorials {
		list = append(list, tutorialSummary{ID: tutorial.ID, Title: tutorial.Title})
	}
	data, err := json.Marshal(list)
	if err != nil {
		return fmt.Errorf("marshal tutorial list: %w", err)
	}
	return ctx.Blob(http.StatusOK, "application/json", data)
}

func (r *Router) GetTutorial(ctx *echo.Context, tutorialID string) error {
	if r.cfg.Tutorials == nil {
		return errNotImplemented()
	}
	tutorial, err := r.cfg.Tutorials.Get(ctx.Request().Context(), tutorialID)
	if errors.Is(err, usecase.ErrNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	}
	if err != nil {
		return fmt.Errorf("get tutorial %q: %w", tutorialID, err)
	}
	data, err := json.Marshal(struct {
		ID             string         `json:"id"`
		Title          string         `json:"title"`
		Description    string         `json:"description"`
		Goal           *string        `json:"goal,omitempty"`
		Lesson         *string        `json:"lesson,omitempty"`
		Guide          []string       `json:"guide,omitempty"`
		InitialProgram jsontext.Value `json:"initialProgram,omitempty"`
		Stage          api.Stage      `json:"stage"`
	}{
		ID: tutorial.ID, Title: tutorial.Title, Description: tutorial.Description,
		Goal: tutorial.Goal, Lesson: tutorial.Lesson, Guide: tutorial.Guide,
		InitialProgram: tutorial.InitialProgram, Stage: tutorial.Stage,
	})
	if err != nil {
		return fmt.Errorf("marshal tutorial %q: %w", tutorialID, err)
	}
	return ctx.Blob(http.StatusOK, "application/json", data)
}

func (r *Router) ListTutorialPrograms(ctx *echo.Context, tutorialID api.TutorialId) error {
	return errNotImplemented()
}

func (r *Router) RunTutorial(ctx *echo.Context, tutorialID api.TutorialId) error {
	return errNotImplemented()
}
