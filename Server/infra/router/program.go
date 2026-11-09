package router

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

func (r *Router) CreateProgram(ctx *echo.Context) error {
	if r.cfg.Programs == nil {
		return errNotImplemented()
	}
	user, ok := UserFromContext(ctx.Request().Context())
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}
	// BodyLimit middleware bounds this read. Unmarshal rejects multiple JSON
	// values, and pointers distinguish missing required fields.
	body, err := io.ReadAll(ctx.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest)).Wrap(fmt.Errorf("read program request: %w", err))
	}
	var request struct {
		ID         *string        `json:"id"`
		Name       *string        `json:"name"`
		Source     jsontext.Value `json:"source"`
		IsBattle   *bool          `json:"isBattle"`
		TutorialID *string        `json:"tutorialID"`
	}
	if err := json.Unmarshal(body, &request, json.RejectUnknownMembers(true)); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest)).Wrap(fmt.Errorf("decode program request: %w", err))
	}
	if request.Name == nil || request.IsBattle == nil {
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}
	// Explicit null is not a valid string for either optional string field.
	var members map[string]jsontext.Value
	if err := json.Unmarshal(body, &members); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest)).Wrap(fmt.Errorf("decode program request members: %w", err))
	}
	for _, key := range []string{"id", "tutorialID"} {
		if value, ok := members[key]; ok && string(value) == "null" {
			return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		}
	}
	program, err := r.cfg.Programs.Create(ctx.Request().Context(), user.GoogleSubject, domain.Program{
		Name: *request.Name, Source: request.Source, IsBattle: *request.IsBattle, TutorialID: request.TutorialID,
	})
	switch {
	case errors.Is(err, domain.ErrInvalidProgram):
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest)).Wrap(fmt.Errorf("create program: %w", err))
	case errors.Is(err, usecase.ErrUnauthenticated):
		return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)).Wrap(fmt.Errorf("create program: %w", err))
	case err != nil:
		return fmt.Errorf("create program: %w", err)
	}
	response, err := json.Marshal(struct {
		ID         string         `json:"id"`
		Name       string         `json:"name"`
		Source     jsontext.Value `json:"source"`
		IsBattle   bool           `json:"isBattle"`
		TutorialID *string        `json:"tutorialID,omitempty"`
	}{program.ID, program.Name, program.Source, program.IsBattle, program.TutorialID})
	if err != nil {
		return fmt.Errorf("marshal created program response: %w", err)
	}
	if err := ctx.JSONBlob(http.StatusCreated, response); err != nil {
		return fmt.Errorf("send created program response: %w", err)
	}
	return nil
}

func (r *Router) CreateProgramWithVibe(ctx *echo.Context) error {
	if r.cfg.ProgramGeneration == nil {
		return errNotImplemented()
	}
	user, ok := UserFromContext(ctx.Request().Context())
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}
	var body jsontext.Value
	if err := json.UnmarshalRead(ctx.Request().Body, &body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest)).Wrap(err)
	}
	var request api.CreateProgramWithVibeJSONBody
	if err := json.Unmarshal(body, &request, json.RejectUnknownMembers(true)); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest)).Wrap(err)
	}
	if request.Prompt == "" {
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}
	var members map[string]jsontext.Value
	if err := json.Unmarshal(body, &members); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest)).Wrap(err)
	}
	for _, key := range []string{"fieldId", "isBattle"} {
		if value, exists := members[key]; exists && string(value) == "null" {
			return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		}
	}
	options := usecase.ProgramGenerationRequest{Prompt: request.Prompt}
	if request.FieldId != nil {
		options.FieldID = *request.FieldId
	}
	if request.IsBattle != nil {
		options.IsBattle = *request.IsBattle
	}
	program, err := r.cfg.ProgramGeneration.GenerateWithOptions(ctx.Request().Context(), user.GoogleSubject, options)
	switch {
	case errors.Is(err, usecase.ErrInvalidGenerationRequest):
		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest)).Wrap(err)
	case errors.Is(err, usecase.ErrUnauthenticated):
		return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)).Wrap(err)
	case errors.Is(err, usecase.ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, http.StatusText(http.StatusNotFound)).Wrap(err)
	case errors.Is(err, usecase.ErrGenerationQuotaExceeded):
		return echo.NewHTTPError(http.StatusTooManyRequests, http.StatusText(http.StatusTooManyRequests)).Wrap(err)
	case errors.Is(err, usecase.ErrInvalidGeneratedProgram):
		return echo.NewHTTPError(http.StatusUnprocessableEntity, http.StatusText(http.StatusUnprocessableEntity)).Wrap(err)
	case errors.Is(err, context.DeadlineExceeded):
		return echo.NewHTTPError(http.StatusGatewayTimeout, http.StatusText(http.StatusGatewayTimeout)).Wrap(err)
	case err != nil:
		return fmt.Errorf("generate program: %w", err)
	}
	return ctx.JSONBlob(http.StatusCreated, program)
}

func (r *Router) GetProgram(ctx *echo.Context, programID api.ProgramId) error {
	return errNotImplemented()
}

func (r *Router) UpdateProgram(ctx *echo.Context, programID api.ProgramId) error {
	return errNotImplemented()
}
