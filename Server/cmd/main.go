package main

import (
	"context"
	"log"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/googleauth"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/static"
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
	var sessions *usecase.Session
	clientID, clientSecret, redirectURL := os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"), os.Getenv("GOOGLE_REDIRECT_URL")
	if (clientID != "" || clientSecret != "" || redirectURL != "") && os.Getenv("DATABASE_URL") == "" {
		log.Fatal("Google login requires DATABASE_URL")
	}
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		pool, err := pgxpool.New(context.Background(), databaseURL)
		if err != nil {
			log.Fatal(err)
		}
		defer pool.Close()
		users := usecase.NewUser(repo.NewUsers(pool))
		sessions = usecase.NewSession(usecase.SessionDeps{Sessions: repo.NewSessions(pool), Users: users})
		cfg.Sessions = sessions
		cfg.Users = users
		cfg.Programs = usecase.NewPrograms(repo.NewPrograms(pool), router.ProgramSourceValidator{})
		if clientID != "" || clientSecret != "" || redirectURL != "" {
			if clientID == "" || clientSecret == "" || redirectURL == "" {
				log.Fatal("GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET and GOOGLE_REDIRECT_URL are all required")
			}
			provider, err := googleauth.New(context.Background(), googleauth.Config{ClientID: clientID, ClientSecret: clientSecret, RedirectURL: redirectURL})
			if err != nil {
				log.Fatal(err)
			}
			cfg.Auth = usecase.NewAuth(usecase.AuthDeps{Provider: googleauth.Provider{Client: provider}, LoginAttempts: repo.NewLoginAttempts(pool), Users: users, Sessions: sessions})
			parsedRedirect, err := url.Parse(redirectURL)
			if err != nil {
				log.Fatal(err)
			}
			cfg.SecureCookies = strings.EqualFold(parsedRedirect.Scheme, "https")
		}
	}
	r := router.NewRouter(cfg)
	authenticate := router.Authentication(sessions)
	api.RegisterHandlersWithOptions(e, r, api.RegisterHandlersOptions{
		BaseURL: "/api",
		OperationMiddlewares: map[string][]echo.MiddlewareFunc{
			"getMe":         {authenticate},
			"logout":        {authenticate},
			"createProgram": {authenticate, middleware.BodyLimit(1 << 20)},
			"setMyUsername": {authenticate},
		},
	})

	if err := e.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
