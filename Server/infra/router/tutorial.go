package router

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"net/http"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
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
	if r.cfg.TutorialRuns == nil {
		return errNotImplemented()
	}
	user, ok := UserFromContext(ctx.Request().Context())
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}
	var request struct {
		ProgramID *string `json:"programId"`
	}
	if err := json.UnmarshalRead(ctx.Request().Body, &request, json.RejectUnknownMembers(true)); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest)).Wrap(fmt.Errorf("decode tutorial run request: %w", err))
	}
	if request.ProgramID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}
	run, err := r.cfg.TutorialRuns.Run(ctx.Request().Context(), user.GoogleSubject, tutorialID, *request.ProgramID)
	status := 0
	switch {
	case errors.Is(err, domain.ErrInvalidProgram):
		status = http.StatusBadRequest
	case errors.Is(err, usecase.ErrUnauthenticated):
		status = http.StatusUnauthorized
	case errors.Is(err, usecase.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, usecase.ErrNotFound):
		status = http.StatusNotFound
	case err != nil:
		return fmt.Errorf("run tutorial: %w", err)
	}
	if status != 0 {
		return echo.NewHTTPError(status, http.StatusText(status)).Wrap(fmt.Errorf("run tutorial: %w", err))
	}
	response, err := json.Marshal(struct {
		RunID   string `json:"runId"`
		FieldID string `json:"fieldId"`
	}{RunID: run.ID, FieldID: run.FieldID})
	if err != nil {
		return fmt.Errorf("marshal tutorial run response: %w", err)
	}
	if err := ctx.JSONBlob(http.StatusCreated, response); err != nil {
		return fmt.Errorf("send tutorial run response: %w", err)
	}
	return nil
}
