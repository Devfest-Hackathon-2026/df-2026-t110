package router

import (
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/labstack/echo/v5"
)

func (r *Router) ListTutorials(ctx *echo.Context) error {
	return errNotImplemented()
}

func (r *Router) GetTutorial(ctx *echo.Context, tutorialID string) error {
	return errNotImplemented()
}

func (r *Router) ListTutorialPrograms(ctx *echo.Context, tutorialID api.TutorialId) error {
	return errNotImplemented()
}
