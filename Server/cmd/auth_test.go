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
