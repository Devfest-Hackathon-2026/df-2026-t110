package main

import (
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/oidcauth"
)

func TestLoginConfigValidate(t *testing.T) {
	t.Parallel()
	complete := loginConfig{Provider: "google", IssuerURL: oidcauth.GoogleIssuer, ClientID: "id", ClientSecret: "secret", RedirectURL: "https://example.com/api/login/callback"}
	for _, tt := range []struct {
		name                       string
		cfg                        loginConfig
		database, enabled, wantErr bool
	}{
		{name: "Google unset stays disabled", cfg: loginConfig{Provider: "google", IssuerURL: oidcauth.GoogleIssuer}},
		{name: "Google configured", cfg: complete, database: true, enabled: true},
		{name: "Google requires database", cfg: complete, enabled: true, wantErr: true},
		{name: "partial Google config fails", cfg: loginConfig{Provider: "google", ClientID: "id"}, database: true, enabled: true, wantErr: true},
		{name: "selected OIDC must be configured", cfg: loginConfig{Provider: "oidc"}, database: true, enabled: true, wantErr: true},
		{name: "OIDC configured", cfg: loginConfig{Provider: "oidc", IssuerURL: "https://issuer.example", ClientID: "id", ClientSecret: "secret", RedirectURL: complete.RedirectURL}, database: true, enabled: true},
		{name: "unknown provider rejected", cfg: loginConfig{Provider: "typo"}, database: true, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.cfg.enabled() != tt.enabled {
				t.Errorf("enabled=%v, want %v", tt.cfg.enabled(), tt.enabled)
			}
			if err := tt.cfg.validate(tt.database); (err != nil) != tt.wantErr {
				t.Errorf("validate error=%v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
func TestReadLoginConfig(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name                     string
		env                      map[string]string
		provider, issuer, client string
		insecure                 bool
	}{
		{name: "Google remains production default", env: map[string]string{"GOOGLE_CLIENT_ID": "google-id", "OIDC_CLIENT_ID": "unused"}, provider: "google", issuer: oidcauth.GoogleIssuer, client: "google-id"},
		{name: "OIDC is explicitly selected", env: map[string]string{"AUTH_PROVIDER": "oidc", "OIDC_ISSUER_URL": "http://dex.localhost:5556/dex", "OIDC_CLIENT_ID": "dex-id", "GOOGLE_CLIENT_ID": "unused", "APP_ENV": "development"}, provider: "oidc", issuer: "http://dex.localhost:5556/dex", client: "dex-id", insecure: true},
		{name: "Cloud Run forbids insecure issuer", env: map[string]string{"AUTH_PROVIDER": "oidc", "APP_ENV": "development", "K_SERVICE": "server"}, provider: "oidc"},
		{name: "production forbids insecure issuer", env: map[string]string{"AUTH_PROVIDER": "oidc", "APP_ENV": "production"}, provider: "oidc"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := readLoginConfig(func(k string) string { return tt.env[k] })
			if cfg.Provider != tt.provider || cfg.IssuerURL != tt.issuer || cfg.ClientID != tt.client || cfg.AllowInsecureHTTP != tt.insecure {
				t.Errorf("config=%+v", cfg)
			}
		})
	}
}
func TestLoginConfigSecureCookies(t *testing.T) {
	t.Parallel()
	for redirect, want := range map[string]bool{"https://example.com/callback": true, "http://localhost:8080/callback": false, "http://127.0.0.1:8080/callback": false} {
		t.Run(redirect, func(t *testing.T) {
			t.Parallel()
			if got := (loginConfig{RedirectURL: redirect}).secureCookies(); got != want {
				t.Errorf("secureCookies=%v,want %v", got, want)
			}
		})
	}
}
