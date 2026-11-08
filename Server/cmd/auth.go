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

// googleLoginConfig is read from the environment. Google sign-in stays disabled
// (the login endpoints answer 501) unless every field is set.
type googleLoginConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

func googleLoginConfigFromEnv() googleLoginConfig {
	return googleLoginConfig{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
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

// secureCookies reports whether auth cookies must be Secure. googleauth.New
// only accepts plain HTTP for localhost, so this follows the redirect scheme.
func (c googleLoginConfig) secureCookies() bool {
	u, err := url.Parse(c.RedirectURL)
	return err == nil && u.Scheme == "https"
}

func newGoogleAuth(ctx context.Context, cfg googleLoginConfig, pool *pgxpool.Pool, users usecase.AuthUsers, sessions usecase.AuthSessions) (*usecase.Auth, error) {
	client, err := googleauth.New(ctx, googleauth.Config{ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret, RedirectURL: cfg.RedirectURL})
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
