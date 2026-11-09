package googleauth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Authorization contains the values needed to complete one sign-in attempt.
// Store it in trusted server-side state and consume it only once within a short
// lifetime; URL is the only field intended to be sent to the browser.
type Authorization struct {
	URL          string
	State        string
	Nonce        string
	CodeVerifier string
}

func identityScopes() []string {
	return []string{oidc.ScopeOpenID}
}

func randomString() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

// Begin generates independent state, nonce, and PKCE values for a sign-in attempt.
func (c *Client) Begin() (Authorization, error) {
	state, err := randomString()
	if err != nil {
		return Authorization{}, fmt.Errorf("generate OAuth state: %w", err)
	}
	nonce, err := randomString()
	if err != nil {
		return Authorization{}, fmt.Errorf("generate OIDC nonce: %w", err)
	}
	verifier, err := randomString()
	if err != nil {
		return Authorization{}, fmt.Errorf("generate PKCE verifier: %w", err)
	}
	return Authorization{
		URL:   c.oauth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)),
		State: state, Nonce: nonce, CodeVerifier: verifier,
	}, nil
}

func equalSecret(a, b string) bool {
	return a != "" && b != "" && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// Complete checks the returned state, exchanges the code with PKCE, and verifies
// the ID token and nonce. The caller must consume its stored Authorization before
// calling Complete to prevent replay. OAuth tokens are discarded because
// sign-in needs only the verified identity.
func (c *Client) Complete(ctx context.Context, attempt Authorization, returnedState, code string) (User, error) {
	if !equalSecret(attempt.State, returnedState) || attempt.Nonce == "" || attempt.CodeVerifier == "" || code == "" {
		return User{}, errors.New("invalid OIDC authorization response")
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	token, err := c.oauth.Exchange(ctx, code, oauth2.VerifierOption(attempt.CodeVerifier))
	if err != nil {
		return User{}, fmt.Errorf("exchange OIDC authorization code: %w", err)
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok || raw == "" {
		return User{}, errors.New("missing OIDC ID token")
	}
	return c.verifyIDToken(ctx, raw, attempt.Nonce, token.AccessToken)
}
