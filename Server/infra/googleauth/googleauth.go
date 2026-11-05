// Package googleauth implements Google sign-in with the OpenID Connect
// authorization code flow.
package googleauth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const googleIssuer = "https://accounts.google.com"

// Config describes a Google OAuth web application and its registered redirect URI.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	HTTPClient   *http.Client
}

// Client signs users in with Google and validates Google ID tokens.
type Client struct {
	oauth      *oauth2.Config
	verifier   *oidc.IDTokenVerifier
	httpClient *http.Client
}

// New discovers Google's OpenID Connect endpoints.
func New(ctx context.Context, cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.ClientID) == "" || strings.TrimSpace(cfg.ClientSecret) == "" {
		return nil, errors.New("a Google OAuth client ID and secret are required")
	}
	if err := validateRedirectURL(cfg.RedirectURL); err != nil {
		return nil, err
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	provider, err := oidc.NewProvider(oidc.ClientContext(ctx, httpClient), googleIssuer)
	if err != nil {
		return nil, fmt.Errorf("discover Google OpenID Connect configuration: %w", err)
	}
	oauth := &oauth2.Config{ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret, RedirectURL: cfg.RedirectURL, Endpoint: provider.Endpoint(), Scopes: identityScopes()}
	return &Client{oauth: oauth, verifier: provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}), httpClient: httpClient}, nil
}

func validateRedirectURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Opaque != "" || u.Fragment != "" {
		return errors.New("invalid Google OAuth redirect URL")
	}
	ip := net.ParseIP(u.Hostname())
	local := strings.EqualFold(u.Hostname(), "localhost") || (ip != nil && ip.IsLoopback())
	if u.Scheme != "https" && (u.Scheme != "http" || !local) {
		return errors.New("redirect URL must use HTTPS except on localhost")
	}
	return nil
}
