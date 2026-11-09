package main

import (
	"context"
	"errors"
	"net/url"
	"os"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/oidcauth"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

type loginConfig struct {
	Provider          string
	IssuerURL         string
	ClientID          string
	ClientSecret      string
	RedirectURL       string
	AllowInsecureHTTP bool
}

func loginConfigFromEnv() loginConfig { return readLoginConfig(os.Getenv) }

func readLoginConfig(getenv func(string) string) loginConfig {
	provider := getenv("AUTH_PROVIDER")
	if provider == "" {
		provider = "google"
	}
	cfg := loginConfig{Provider: provider}
	if provider == "oidc" {
		cfg.IssuerURL = getenv("OIDC_ISSUER_URL")
		cfg.ClientID = getenv("OIDC_CLIENT_ID")
		cfg.ClientSecret = getenv("OIDC_CLIENT_SECRET")
		cfg.RedirectURL = getenv("OIDC_REDIRECT_URL")
		cfg.AllowInsecureHTTP = getenv("APP_ENV") == "development" && getenv("K_SERVICE") == ""
	} else {
		cfg.IssuerURL = oidcauth.GoogleIssuer
		cfg.ClientID = getenv("GOOGLE_CLIENT_ID")
		cfg.ClientSecret = getenv("GOOGLE_CLIENT_SECRET")
		cfg.RedirectURL = getenv("GOOGLE_REDIRECT_URL")
	}
	return cfg
}

func (c loginConfig) enabled() bool {
	return c.Provider == "oidc" || c.ClientID != "" || c.ClientSecret != "" || c.RedirectURL != ""
}

func (c loginConfig) validate(hasDatabase bool) error {
	if c.Provider != "google" && c.Provider != "oidc" {
		return errors.New("AUTH_PROVIDER must be google or oidc")
	}
	if !c.enabled() {
		return nil
	}
	if c.IssuerURL == "" || c.ClientID == "" || c.ClientSecret == "" || c.RedirectURL == "" {
		return errors.New("the selected auth provider requires an issuer, client ID, client secret and redirect URL")
	}
	if !hasDatabase {
		return errors.New("login requires DATABASE_URL to store login attempts and sessions")
	}
	return nil
}

func (c loginConfig) secureCookies() bool {
	u, err := url.Parse(c.RedirectURL)
	return err == nil && u.Scheme == "https"
}

func newLoginAuth(ctx context.Context, cfg loginConfig, pool *pgxpool.Pool, users usecase.AuthUsers, sessions usecase.AuthSessions) (*usecase.Auth, error) {
	client, err := oidcauth.New(ctx, oidcauth.Config{IssuerURL: cfg.IssuerURL, ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret, RedirectURL: cfg.RedirectURL, AllowInsecureHTTP: cfg.AllowInsecureHTTP})
	if err != nil {
		return nil, err
	}
	return usecase.NewAuth(usecase.AuthDeps{
		Provider:      oidcauth.Provider{Client: client},
		LoginAttempts: repo.NewLoginAttempts(pool),
		Users:         users,
		Sessions:      sessions,
	}), nil
}
