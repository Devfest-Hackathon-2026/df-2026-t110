// Package googleauth implements sign-in with the OpenID Connect authorization code flow.
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

// GoogleIssuer is the default provider for existing production deployments.
const GoogleIssuer = "https://accounts.google.com"

// Config describes an OIDC relying party and its registered redirect URI.
type Config struct {
	IssuerURL         string
	ClientID          string
	ClientSecret      string
	RedirectURL       string
	HTTPClient        *http.Client
	AllowInsecureHTTP bool
}

// Client signs users in with a configured issuer and verifies its ID tokens.
type Client struct {
	issuer     string
	oauth      *oauth2.Config
	verifier   *oidc.IDTokenVerifier
	httpClient *http.Client
}

// New discovers the issuer's endpoints and retains full issuer validation.
func New(ctx context.Context, cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.ClientID) == "" || strings.TrimSpace(cfg.ClientSecret) == "" {
		return nil, errors.New("an OIDC client ID and secret are required")
	}
	issuer := cfg.IssuerURL
	if issuer == "" {
		issuer = GoogleIssuer
	}
	if err := validateIssuerURL(issuer, cfg.AllowInsecureHTTP); err != nil {
		return nil, err
	}
	if err := validateRedirectURL(cfg.RedirectURL); err != nil {
		return nil, err
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	provider, err := oidc.NewProvider(oidc.ClientContext(ctx, httpClient), issuer)
	if err != nil {
		return nil, fmt.Errorf("discover OpenID Connect configuration: %w", err)
	}
	oauth := &oauth2.Config{ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret, RedirectURL: cfg.RedirectURL, Endpoint: provider.Endpoint(), Scopes: identityScopes()}
	return &Client{issuer: issuer, oauth: oauth, verifier: provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}), httpClient: httpClient}, nil
}

func validateRedirectURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Opaque != "" || u.Fragment != "" {
		return errors.New("invalid OAuth redirect URL")
	}
	ip := net.ParseIP(u.Hostname())
	local := strings.EqualFold(u.Hostname(), "localhost") || strings.HasSuffix(strings.ToLower(u.Hostname()), ".localhost") || (ip != nil && ip.IsLoopback())
	if u.Scheme != "https" && (u.Scheme != "http" || !local) {
		return errors.New("URL must use HTTPS except on localhost")
	}
	return nil
}

func validateIssuerURL(raw string, allowHTTP bool) error {
	u, err := url.Parse(raw)
	if err != nil || u.RawQuery != "" || u.ForceQuery || validateRedirectURL(raw) != nil {
		return errors.New("invalid OIDC issuer URL")
	}
	if u.Scheme != "https" && !allowHTTP {
		return errors.New("HTTP OIDC issuer requires local development mode")
	}
	return nil
}
