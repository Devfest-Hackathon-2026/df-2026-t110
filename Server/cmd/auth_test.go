package main

import "testing"

func TestGoogleLoginConfigValidate(t *testing.T) {
	t.Parallel()

	complete := googleLoginConfig{ClientID: "id", ClientSecret: "secret", RedirectURL: "https://example.com/api/login/callback"}
	tests := []struct {
		name        string
		cfg         googleLoginConfig
		hasDatabase bool
		wantEnabled bool
		wantErr     bool
	}{
		{name: "unset stays disabled without a database", cfg: googleLoginConfig{}, wantEnabled: false},
		{name: "complete with database", cfg: complete, hasDatabase: true, wantEnabled: true},
		{name: "complete without database", cfg: complete, wantEnabled: true, wantErr: true},
		{name: "missing secret", cfg: googleLoginConfig{ClientID: "id", RedirectURL: complete.RedirectURL}, hasDatabase: true, wantEnabled: true, wantErr: true},
		{name: "only redirect URL", cfg: googleLoginConfig{RedirectURL: complete.RedirectURL}, hasDatabase: true, wantEnabled: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.cfg.enabled(); got != tt.wantEnabled {
				t.Errorf("enabled() = %v, want %v", got, tt.wantEnabled)
			}
			if err := tt.cfg.validate(tt.hasDatabase); (err != nil) != tt.wantErr {
				t.Errorf("validate(%v) error = %v, wantErr %v", tt.hasDatabase, err, tt.wantErr)
			}
		})
	}
}

func TestGoogleLoginConfigSecureCookies(t *testing.T) {
	t.Parallel()

	tests := map[string]bool{
		"https://example.com/api/login/callback":   true,
		"http://localhost:5173/api/login/callback": false,
		"http://127.0.0.1:8080/api/login/callback": false,
	}
	for redirectURL, want := range tests {
		t.Run(redirectURL, func(t *testing.T) {
			t.Parallel()

			if got := (googleLoginConfig{RedirectURL: redirectURL}).secureCookies(); got != want {
				t.Errorf("secureCookies() = %v, want %v", got, want)
			}
		})
	}
}

func TestReadGoogleLoginConfig(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name                               string
		env                                map[string]string
		issuer, clientID, secret, redirect string
		enabled, insecure                  bool
	}{
		{name: "Google remains the default", issuer: "https://accounts.google.com"},
		{name: "existing Google parameters", env: map[string]string{"GOOGLE_CLIENT_ID": "google-id", "GOOGLE_CLIENT_SECRET": "google-secret", "GOOGLE_REDIRECT_URL": "https://app.example/api/login/callback"}, issuer: "https://accounts.google.com", clientID: "google-id", secret: "google-secret", redirect: "https://app.example/api/login/callback", enabled: true},
		{name: "Dex parameters injected through Google settings", env: map[string]string{"GOOGLE_ISSUER_URL": "http://dex.localhost:5556/dex", "GOOGLE_CLIENT_ID": "dex-id", "GOOGLE_CLIENT_SECRET": "dex-secret", "GOOGLE_REDIRECT_URL": "http://localhost:8080/api/login/callback", "APP_ENV": "development"}, issuer: "http://dex.localhost:5556/dex", clientID: "dex-id", secret: "dex-secret", redirect: "http://localhost:8080/api/login/callback", enabled: true, insecure: true},
		{name: "Cloud Run forbids insecure issuer", env: map[string]string{"APP_ENV": "development", "K_SERVICE": "server"}, issuer: "https://accounts.google.com"},
		{name: "production forbids insecure issuer", env: map[string]string{"APP_ENV": "production"}, issuer: "https://accounts.google.com"},
		{name: "no separate provider selector", env: map[string]string{"AUTH_PROVIDER": "oidc", "OIDC_ISSUER_URL": "https://unused.example", "OIDC_CLIENT_ID": "unused"}, issuer: "https://accounts.google.com"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := readGoogleLoginConfig(func(k string) string { return tt.env[k] })
			if cfg.IssuerURL != tt.issuer || cfg.ClientID != tt.clientID || cfg.ClientSecret != tt.secret || cfg.RedirectURL != tt.redirect || cfg.enabled() != tt.enabled || cfg.AllowInsecureHTTP != tt.insecure {
				t.Errorf("config=%+v", cfg)
			}
		})
	}
}
