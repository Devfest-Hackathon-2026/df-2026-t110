package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/gemini"
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
	"golang.org/x/oauth2/google"
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
		cfg.Matches = usecase.NewMatches(usecase.MatchDeps{
			Users: users, Programs: programs, Opponents: matches, Layouts: layouts,
			Executor: matchrun.NewExecutor(), Matches: matches,
		})
		runs := usecase.NewTutorialRuns(programs, textbooks, tutorialrun.NewExecutor(), repo.NewRuns(pool))
		cfg.TutorialRuns = runs
		cfg.RunResults = usecase.NewRunResults(repo.NewRuns(pool))
		if project := os.Getenv("GOOGLE_CLOUD_PROJECT"); project != "" {
			location := os.Getenv("GENERATION_LOCATION")
			if location == "" {
				location = "global"
			}
			model := os.Getenv("GENERATION_MODEL")
			if model == "" {
				model = "gemini-3.5-flash"
			}
			client, err := google.DefaultClient(context.Background(), "https://www.googleapis.com/auth/cloud-platform")
			if err != nil {
				log.Fatal(err)
			}
			generator, err := gemini.NewGenerator(client, project, location, model)
			if err != nil {
				log.Fatal(err)
			}
			quota, err := usecase.NewGenerationTokenQuota(repo.NewGenerationUsages(pool), 100_000, 24*time.Hour, nil)
			if err != nil {
				log.Fatal(err)
			}
			cfg.ProgramGeneration = usecase.NewProgramGeneration(usecase.ProgramGenerationDeps{
				Fields: textbooks, Quota: quota, Generator: generator,
				Validator: router.GeneratedProgramValidator{}, MaxTokens: 8_192, Timeout: 60 * time.Second,
			})
		}
	}
	r := router.NewRouter(cfg)
	authenticate := router.Authentication(sessions)
	api.RegisterHandlersWithOptions(e, r, api.RegisterHandlersOptions{
		BaseURL: "/api",
		OperationMiddlewares: map[string][]echo.MiddlewareFunc{
			"createMatch": {authenticate, middleware.BodyLimit(1 << 20)},
			"getBattlePrograms":     {authenticate},
			"setBattlePrograms":     {authenticate, middleware.BodyLimit(1 << 20)},
			"getMe":                 {authenticate},
			"logout":                {authenticate},
			"createProgram":         {authenticate, middleware.BodyLimit(1 << 20)},
			"createProgramWithVibe": {authenticate, middleware.BodyLimit(16 << 10)},
			"runTutorial":           {authenticate, middleware.BodyLimit(1 << 20)},
			"listTutorialPrograms":  {authenticate},
			"runEvents":             {authenticate},
			"setMyUsername":         {authenticate},
		},
	})

	if err := e.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
