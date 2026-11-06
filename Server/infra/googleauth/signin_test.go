package googleauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCompleteRejectsInvalidStateWithoutExchange(t *testing.T) {
	f := newProviderFixture(t)
	a := f.begin(t)
	if _, err := f.client.Complete(context.Background(), a, "wrong-state", "valid-code"); err == nil {
		t.Fatal("wrong state accepted")
	}
	if f.exchanges != 0 {
		t.Fatal("exchanged code with invalid state")
	}
}

func TestBeginRequestsOnlyIdentity(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != googleIssuer+"/.well-known/openid-configuration" {
			return nil, fmt.Errorf("unexpected discovery URL: %s", r.URL)
		}
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{ // #nosec G101 -- These discovery endpoints are public URLs, not credentials.
			"issuer":                 googleIssuer,
			"authorization_endpoint": "https://accounts.google.com/authorize",
			"token_endpoint":         "https://accounts.google.com/token",
			"jwks_uri":               "https://accounts.google.com/keys",
		})
		return w.Result(), nil
	})}
	client, err := New(context.Background(), Config{ClientID: "client-id", ClientSecret: "client-secret", RedirectURL: "http://localhost:8080/callback", HTTPClient: httpClient})
	if err != nil {
		t.Fatal(err)
	}
	a, err := client.Begin()
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(a.URL)
	if err != nil {
		t.Fatal(err)
	}
	if q := u.Query(); q.Get("scope") != "openid" || q.Has("access_type") {
		t.Fatalf("authorization URL requests more than identity: %s", u.RawQuery)
	}
}
