package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/matchrun"
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
	fields := usecase.NewFields(textbooks)
	cfg := router.Config{Tutorials: usecase.NewTutorials(textbooks, nil, nil), Fields: fields}
	var sessions router.SessionUserLookup
	googleLogin := googleLoginConfigFromEnv()
	if err := googleLogin.validate(os.Getenv("DATABASE_URL") != ""); err != nil {
		log.Fatal(err)
	}
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		pool, err := pgxpool.New(context.Background(), databaseURL)
		if err != nil {
			log.Fatal(err)
		}
		defer pool.Close()
		users := usecase.NewUser(repo.NewUsers(pool))
		sessionService := usecase.NewSession(usecase.SessionDeps{Sessions: repo.NewSessions(pool), Users: users})
		sessions = sessionService
		cfg.Sessions = sessionService
		cfg.Users = users
		if googleLogin.enabled() {
			auth, err := newGoogleAuth(context.Background(), googleLogin, pool, users, sessionService)
			if err != nil {
				log.Fatal(err)
			}
			cfg.Auth = auth
			cfg.SecureCookies = googleLogin.secureCookies()
		}
		programs := repo.NewPrograms(pool)
		cfg.Programs = usecase.NewPrograms(programs, router.ProgramSourceValidator{})
		cfg.BattlePrograms = usecase.NewBattlePrograms(repo.NewBattlePrograms(pool), programs)
		matches := repo.NewMatches(pool)
		layouts, err := matchrun.NewLayouts()
		if err != nil {
			log.Fatal(err)
		}
		runExecutor := usecase.NewExecutor(repo.NewRuns(pool), matches, tutorialrun.Simulator{}, matchrun.Simulator{})
		cfg.Matches = usecase.NewMatches(usecase.MatchDeps{
			Users: users, Programs: programs, Opponents: matches, Layouts: layouts,
			Executor: runExecutor, Matches: matches,
		})
		cfg.Rankings = usecase.NewRankings(matches)
		cfg.Tutorials = usecase.NewTutorials(textbooks, programs, runExecutor)
		cfg.RunResults = usecase.NewRunResults(repo.NewRuns(pool))
	}
	r := router.NewRouter(cfg)
	authenticate := router.Authentication(sessions)
	api.RegisterHandlersWithOptions(e, r, api.RegisterHandlersOptions{
		BaseURL: "/api",
		OperationMiddlewares: map[string][]echo.MiddlewareFunc{
			"createMatch":          {authenticate, middleware.BodyLimit(1 << 20)},
			"getBattlePrograms":    {authenticate},
			"setBattlePrograms":    {authenticate, middleware.BodyLimit(1 << 20)},
			"getMe":                {authenticate},
			"logout":               {authenticate},
			"createProgram":        {authenticate, middleware.BodyLimit(1 << 20)},
			"getProgram":           {authenticate},
			"updateProgram":        {authenticate, middleware.BodyLimit(1 << 20)},
			"runTutorial":          {authenticate, middleware.BodyLimit(1 << 20)},
			"listTutorialPrograms": {authenticate},
			"runEvents":            {authenticate},
			"setMyUsername":        {authenticate},
		},
	})

	if err := e.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
