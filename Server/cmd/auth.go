package main

import (
	"context"
	"errors"
	"net/url"
	"os"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/googleauth"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Google remains the default issuer. Local development can inject an OIDC test
// issuer through the same Google login configuration.
type googleLoginConfig struct {
	IssuerURL         string
	ClientID          string
	ClientSecret      string
	RedirectURL       string
	AllowInsecureHTTP bool
}

func googleLoginConfigFromEnv() googleLoginConfig { return readGoogleLoginConfig(os.Getenv) }

func readGoogleLoginConfig(getenv func(string) string) googleLoginConfig {
	issuer := getenv("GOOGLE_ISSUER_URL")
	if issuer == "" {
		issuer = googleauth.GoogleIssuer
	}
	return googleLoginConfig{
		IssuerURL:         issuer,
		ClientID:          getenv("GOOGLE_CLIENT_ID"),
		ClientSecret:      getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:       getenv("GOOGLE_REDIRECT_URL"),
		AllowInsecureHTTP: getenv("APP_ENV") == "development" && getenv("K_SERVICE") == "",
	}
}

func (c googleLoginConfig) enabled() bool {
	return c.ClientID != "" || c.ClientSecret != "" || c.RedirectURL != ""
}

func (c googleLoginConfig) validate(hasDatabase bool) error {
	if !c.enabled() {
		return nil
	}
	if c.ClientID == "" || c.ClientSecret == "" || c.RedirectURL == "" {
		return errors.New("GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET and GOOGLE_REDIRECT_URL must be set together")
	}
	if !hasDatabase {
		return errors.New("google login requires DATABASE_URL to store login attempts and sessions")
	}
	return nil
}

func (c googleLoginConfig) secureCookies() bool {
	u, err := url.Parse(c.RedirectURL)
	return err == nil && u.Scheme == "https"
}

func newGoogleAuth(ctx context.Context, cfg googleLoginConfig, pool *pgxpool.Pool, users usecase.AuthUsers, sessions usecase.AuthSessions) (*usecase.Auth, error) {
	client, err := googleauth.New(ctx, googleauth.Config{
		IssuerURL:         cfg.IssuerURL,
		ClientID:          cfg.ClientID,
		ClientSecret:      cfg.ClientSecret,
		RedirectURL:       cfg.RedirectURL,
		AllowInsecureHTTP: cfg.AllowInsecureHTTP,
	})
	if err != nil {
		return nil, err
	}
	return usecase.NewAuth(usecase.AuthDeps{
		Provider:      googleauth.Provider{Client: client},
		LoginAttempts: repo.NewLoginAttempts(pool),
		Users:         users,
		Sessions:      sessions,
	}), nil
}
