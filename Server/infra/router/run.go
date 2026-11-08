package router

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

func (r *Router) RunEvents(ctx *echo.Context, runID api.RunId) error {
	if r.cfg.RunResults == nil {
		return errNotImplemented()
	}
	user, ok := UserFromContext(ctx.Request().Context())
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}
	requestContext := ctx.Request().Context()

	// Check ownership and existence before opening a successful SSE response.
	result, err := r.cfg.RunResults.Result(requestContext, user.GoogleSubject, runID.String())
	switch {
	case errors.Is(err, usecase.ErrUnauthenticated):
		return echo.NewHTTPError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)).Wrap(err)
	case errors.Is(err, usecase.ErrForbidden):
		return echo.NewHTTPError(http.StatusForbidden, http.StatusText(http.StatusForbidden)).Wrap(err)
	case errors.Is(err, usecase.ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, http.StatusText(http.StatusNotFound)).Wrap(err)
	case err != nil:
		return fmt.Errorf("get run result: %w", err)
	}
	response := ctx.Response()
	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-store")

	// Completed runs skip waiting. For pending runs, keep the connection open
	// until the DB contains a result or the client disconnects.
	if len(result) == 0 {
		if err := http.NewResponseController(response).Flush(); err != nil {
			return fmt.Errorf("start run event stream: %w", err)
		}
		result, err = r.waitForRunResult(requestContext, user.GoogleSubject, runID.String())
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return nil
		case err != nil:
			return fmt.Errorf("wait for run result: %w", err)
		}
	}

	// Send the complete result once, then close the stream by returning.
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode run result: %w", err)
	}
	if _, err := fmt.Fprintf(response, "event: result\ndata: %s\n\n", data); err != nil {
		return fmt.Errorf("write run result event: %w", err)
	}
	return http.NewResponseController(response).Flush()
}

func (r *Router) waitForRunResult(ctx context.Context, subject, runID string) (jsontext.Value, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			result, err := r.cfg.RunResults.Result(ctx, subject, runID)
			if err != nil {
				return nil, err
			}
			if len(result) != 0 {
				return result, nil
			}
		}
	}
}
