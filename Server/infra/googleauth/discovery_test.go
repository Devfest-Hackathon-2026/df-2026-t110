package googleauth

import (
	"crypto"
	"encoding/base64"
	json "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	jose "github.com/go-jose/go-jose/v4"
)

func TestConfiguredIssuerDiscoveryAndIdentity(t *testing.T) {
	t.Parallel()
	fixture := newProviderFixture(t)
	var issuer string
	discovery := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.MarshalWrite(w, map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/auth", "token_endpoint": fixture.provider.URL, "jwks_uri": issuer + "/keys"})
		case "/keys":
			_ = json.MarshalWrite(w, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &fixture.key.PublicKey, Use: "sig", Algorithm: "RS256"}}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(discovery.Close)
	issuer = discovery.URL
	cfg := Config{IssuerURL: issuer, ClientID: "client-id", ClientSecret: "client-secret", RedirectURL: "http://localhost:8080/callback", AllowInsecureHTTP: true, HTTPClient: discovery.Client()}
	client, err := New(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	provider := Provider{Client: client}
	request, err := provider.Begin()
	if err != nil {
		t.Fatal(err)
	}
	auth, err := url.Parse(request.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(request.URL, issuer+"/auth?") {
		t.Errorf("authorization URL=%s", request.URL)
	}
	fixture.challenge = auth.Query().Get("code_challenge")
	payload := claims(request.Nonce)
	payload["iss"] = issuer
	fixture.idToken = fixture.sign(t, payload)
	identity, err := provider.Complete(t.Context(), request, request.State, "valid-code")
	if err != nil {
		t.Fatal(err)
	}
	want := "oidc:" + base64.RawURLEncoding.EncodeToString([]byte(issuer)) + ":google-subject"
	if identity.Subject != want {
		t.Errorf("subject=%q,want %q", identity.Subject, want)
	}
	// Issuer verification remains active when the local HTTP transport is enabled.
	payload["iss"] = "https://different.example"
	fixture.idToken = fixture.sign(t, payload)
	if _, err := provider.Complete(t.Context(), request, request.State, "valid-code"); err == nil {
		t.Error("accepted a token from another issuer")
	}
}

func TestDiscoveryRejectsIssuerMismatch(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.MarshalWrite(w, map[string]string{"issuer": "https://different.example"})
	}))
	t.Cleanup(server.Close)
	if _, err := New(t.Context(), Config{IssuerURL: server.URL, ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost:8080/callback", AllowInsecureHTTP: true, HTTPClient: server.Client()}); err == nil {
		t.Error("accepted discovery with a different issuer")
	}
}

func TestIssuerURLValidation(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		url          string
		allow, valid bool
	}{
		{"https://issuer.example/oidc", false, true},
		{"http://dex.localhost:5556/dex", true, true},
		{"http://127.0.0.1:5556/dex", true, true},
		{"http://dex.localhost:5556/dex", false, false},
		{"http://issuer.example", true, false},
		{"https://issuer.example?query=value", false, false},
		{"https://issuer.example#fragment", false, false},
		{"https://user@issuer.example", false, false},
		{"/relative", true, false},
	} {
		t.Run(tt.url, func(t *testing.T) {
			t.Parallel()
			if err := validateIssuerURL(tt.url, tt.allow); (err == nil) != tt.valid {
				t.Errorf("valid=%v,error=%v", tt.valid, err)
			}
		})
	}
}

// Same subjects from two issuers must not resolve to the same stored user.
func TestProviderSeparatesIssuerSubjects(t *testing.T) {
	t.Parallel()
	subjects := make(map[string]struct{})
	for _, issuer := range []string{GoogleIssuer, "https://dex.example", "https://authentik.example/application/o/app/"} {
		f := newProviderFixture(t)
		f.client.issuer = issuer
		f.client.verifier = oidc.NewVerifier(issuer, &oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{&f.key.PublicKey}}, &oidc.Config{ClientID: "client-id"})
		provider := Provider{Client: f.client}
		request, err := provider.Begin()
		if err != nil {
			t.Fatal(err)
		}
		u, err := url.Parse(request.URL)
		if err != nil {
			t.Fatal(err)
		}
		f.challenge = u.Query().Get("code_challenge")
		payload := claims(request.Nonce)
		payload["iss"] = issuer
		f.idToken = f.sign(t, payload)
		identity, err := provider.Complete(t.Context(), request, request.State, "valid-code")
		if err != nil {
			t.Fatal(err)
		}
		if _, exists := subjects[identity.Subject]; exists {
			t.Errorf("identity collision for %s", issuer)
		}
		subjects[identity.Subject] = struct{}{}
	}
}
