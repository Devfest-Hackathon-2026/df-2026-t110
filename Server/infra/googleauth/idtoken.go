package googleauth

import (
	"context"
	"errors"
	"fmt"
)

// User identifies an account by Google's stable OIDC subject.
type User struct {
	ID   string
	Name string
}

func (c *Client) verifyIDToken(ctx context.Context, raw, nonce, accessToken string) (User, error) {
	idToken, err := c.verifier.Verify(ctx, raw)
	if err != nil {
		return User{}, fmt.Errorf("verify Google ID token: %w", err)
	}
	if idToken.Subject == "" || !equalSecret(idToken.Nonce, nonce) {
		return User{}, errors.New("invalid Google identity or nonce")
	}
	if idToken.AccessTokenHash != "" {
		if err := idToken.VerifyAccessToken(accessToken); err != nil {
			return User{}, fmt.Errorf("verify Google access token binding: %w", err)
		}
	}
	var claims struct {
		Name            string `json:"name"`
		AuthorizedParty string `json:"azp"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return User{}, fmt.Errorf("read Google ID token claims: %w", err)
	}
	if (len(idToken.Audience) > 1 && claims.AuthorizedParty == "") || (claims.AuthorizedParty != "" && claims.AuthorizedParty != c.oauth.ClientID) {
		return User{}, errors.New("invalid Google ID token authorized party")
	}
	return User{ID: idToken.Subject, Name: claims.Name}, nil
}
