package router

import (
	"context"
	"encoding/json/jsontext"
	"net/http"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

var _ api.ServerInterface = (*Router)(nil)

// Config holds HTTP-level settings of the handlers.
type Config struct {
	Auth              AuthService
	Sessions          SessionService
	SecureCookies     bool
	Users             UsernameSetter
	Programs          ProgramCreator
	ProgramGeneration ProgramGenerationService
	Tutorials         TutorialService
	TutorialRuns      TutorialRunner
	RunResults        RunResultGetter
}

type AuthService interface {
	BeginLogin(ctx context.Context) (usecase.LoginAttempt, error)
	CompleteLogin(ctx context.Context, callback usecase.LoginCallback) (domain.Session, error)
}

type SessionService interface {
	SessionUserLookup
	Logout(ctx context.Context, sessionID string) error
}

type TutorialService interface {
	List(ctx context.Context) ([]domain.Tutorial, error)
	Get(ctx context.Context, id string) (domain.Tutorial, error)
}

type RunResultGetter interface {
	Result(ctx context.Context, subject, runID string) (jsontext.Value, error)
}

type TutorialRunner interface {
	Run(ctx context.Context, subject, tutorialID, programID string) (domain.TutorialRun, error)
}

type ProgramService interface {
	Create(ctx context.Context, subject string, draft domain.Program) (domain.Program, error)
	ListByTutorialID(ctx context.Context, subject, tutorialID string) ([]domain.Program, error)
}

type ProgramGenerationService interface {
	GenerateWithOptions(ctx context.Context, subject string, request usecase.ProgramGenerationRequest) (jsontext.Value, error)
}

type UsernameSetter interface {
	SetUsername(ctx context.Context, subject, name string) (domain.User, error)
}

// Router translates HTTP requests into use case calls.
type Router struct {
	cfg Config
}

func NewRouter(cfg Config) *Router {
	return &Router{cfg: cfg}
}

func errNotImplemented() error {
	return echo.NewHTTPError(http.StatusNotImplemented, http.StatusText(http.StatusNotImplemented))
}
