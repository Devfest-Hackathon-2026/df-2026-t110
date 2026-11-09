package oidcauth

import (
	"context"
	"net/url"
	"testing"
)

func TestProviderCompletesStoredRequest(t *testing.T) {
	t.Parallel()

	f := newProviderFixture(t)
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
	f.idToken = f.sign(t, claims(request.Nonce))

	// The login attempt store keeps everything except the URL.
	stored := request
	stored.URL = ""
	identity, err := provider.Complete(context.Background(), stored, request.State, "valid-code")
	if err != nil {
		t.Fatal(err)
	}
	if identity.Subject != "google-subject" {
		t.Errorf("Subject = %q, want google-subject", identity.Subject)
	}
}
