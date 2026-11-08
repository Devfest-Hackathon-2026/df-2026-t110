package router

import (
	"log/slog"

	"github.com/labstack/echo/v5"
)

// HTTPErrorHandler logs the complete error chain while keeping internal details
// out of the public response. Route templates avoid logging query parameters or
// path parameter values. Errors after a response was sent are still logged.
func HTTPErrorHandler(logger *slog.Logger) echo.HTTPErrorHandler {
	respond := echo.DefaultHTTPErrorHandler(false)
	return func(ctx *echo.Context, err error) {
		_, status := echo.ResolveResponseStatus(ctx.Response(), err)
		logger.ErrorContext(ctx.Request().Context(), "HTTP request failed",
			slog.String("method", ctx.Request().Method),
			slog.String("route", ctx.Path()),
			slog.Int("status", status),
			slog.Any("error", err),
		)
		respond(ctx, err)
	}
}
