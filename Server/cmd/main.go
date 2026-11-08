package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/static"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/tutorialrun"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	e := echo.NewWithConfig(echo.Config{
		Logger:           logger,
		HTTPErrorHandler: router.HTTPErrorHandler(logger),
	})

	// Unset in local development, where the Vite dev server serves the client and proxies /api.
	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		e.Use(static.Middleware(dir, "/api"))
	}

	textbooks, err := repo.NewTextbooks(context.Background(), textbookPages())
	if err != nil {
		log.Fatal(err)
	}
	cfg := router.Config{Tutorials: textbooks}
	var sessions router.SessionUserLookup
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		pool, err := pgxpool.New(context.Background(), databaseURL)
		if err != nil {
			log.Fatal(err)
		}
		defer pool.Close()
		users := usecase.NewUser(repo.NewUsers(pool))
		sessions = usecase.NewSession(usecase.SessionDeps{Sessions: repo.NewSessions(pool), Users: users})
		cfg.Users = users
		programs := repo.NewPrograms(pool)
		cfg.Programs = usecase.NewPrograms(programs, router.ProgramSourceValidator{})
		runs := usecase.NewTutorialRuns(programs, textbooks, tutorialrun.NewExecutor(), repo.NewRuns(pool))
		cfg.TutorialRuns = runs
	}
	r := router.NewRouter(cfg)
	authenticate := router.Authentication(sessions)
	api.RegisterHandlersWithOptions(e, r, api.RegisterHandlersOptions{
		BaseURL: "/api",
		OperationMiddlewares: map[string][]echo.MiddlewareFunc{
			"createProgram": {authenticate, middleware.BodyLimit(1 << 20)},
			"runTutorial":   {authenticate, middleware.BodyLimit(1 << 20)},
			"setMyUsername": {authenticate},
		},
	})

	if err := e.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
