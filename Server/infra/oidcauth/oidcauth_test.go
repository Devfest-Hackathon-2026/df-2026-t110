package oidcauth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	json "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	jose "github.com/go-jose/go-jose/v4"
	"golang.org/x/oauth2"
)

type providerFixture struct {
	client    *Client
	provider  *httptest.Server
	key       *rsa.PrivateKey
	idToken   string
	challenge string
	exchanges int
}

func newProviderFixture(t *testing.T) *providerFixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &providerFixture{key: key}
	f.provider = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		f.exchanges++
		sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
		if r.Form.Get("code") != "valid-code" || f.challenge != base64.RawURLEncoding.EncodeToString(sum[:]) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = json.MarshalWrite(w, map[string]any{"access_token": "access-token", "refresh_token": "refresh-token", "token_type": "Bearer", "expires_in": 3600, "id_token": f.idToken, "scope": "openid"})
	}))
	t.Cleanup(f.provider.Close)
	oauth := &oauth2.Config{ClientID: "client-id", ClientSecret: "client-secret", RedirectURL: "http://localhost:8080/callback", Scopes: identityScopes(), Endpoint: oauth2.Endpoint{AuthURL: f.provider.URL + "/authorize", TokenURL: f.provider.URL + "/token", AuthStyle: oauth2.AuthStyleInParams}}
	verifier := oidc.NewVerifier(GoogleIssuer, &oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{&key.PublicKey}}, &oidc.Config{ClientID: "client-id"})
	f.client = &Client{issuer: GoogleIssuer, oauth: oauth, verifier: verifier, httpClient: f.provider.Client()}
	return f
}

func (f *providerFixture) begin(t *testing.T) Authorization {
	t.Helper()
	a, err := f.client.Begin()
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(a.URL)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if a.State == "" || a.Nonce == "" || a.CodeVerifier == "" || q.Get("state") != a.State || q.Get("nonce") != a.Nonce || q.Get("response_type") != "code" || q.Get("code_challenge_method") != "S256" {
		t.Fatalf("invalid authorization URL: %s", a.URL)
	}
	f.challenge = q.Get("code_challenge")
	return a
}

func (f *providerFixture) sign(t *testing.T, claims map[string]any) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: f.key}, nil)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := signed.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func claims(nonce string) map[string]any {
	return map[string]any{"iss": GoogleIssuer, "aud": "client-id", "sub": "google-subject", "name": "Alice", "nonce": nonce, "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix()}
}

func TestAuthorizationAndIdentity(t *testing.T) {
	t.Parallel()

	f := newProviderFixture(t)
	a := f.begin(t)
	f.idToken = f.sign(t, claims(a.Nonce))
	user, err := f.client.Complete(context.Background(), a, a.State, "valid-code")
	if err != nil {
		t.Fatal(err)
	}
	if user != (User{Subject: "google-subject"}) {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestRedirectURLValidation(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		url   string
		valid bool
	}{
		{"https://example.com/any/callback", true},
		{"http://localhost:8080/callback", true},
		{"http://127.0.0.1:8080/callback", true},
		{"http://example.com/callback", false},
		{"https://example.com/callback#fragment", false},
		{"https://user@example.com/callback", false},
		{"/relative", false},
	} {
		if err := validateRedirectURL(tt.url); (err == nil) != tt.valid {
			t.Errorf("URL %q: valid=%v, error=%v", tt.url, tt.valid, err)
		}
	}
}
