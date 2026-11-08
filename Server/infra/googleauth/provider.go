package googleauth

import (
	"context"

	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

// Provider adapts the Google OIDC client to the authentication use case.
type Provider struct{ Client *Client }

func (p Provider) Begin() (usecase.AuthorizationRequest, error) {
	a, err := p.Client.Begin()
	return usecase.AuthorizationRequest{URL: a.URL, State: a.State, Nonce: a.Nonce, CodeVerifier: a.CodeVerifier}, err
}

func (p Provider) Complete(ctx context.Context, request usecase.AuthorizationRequest, state, code string) (usecase.Identity, error) {
	user, err := p.Client.Complete(ctx, Authorization{URL: request.URL, State: request.State, Nonce: request.Nonce, CodeVerifier: request.CodeVerifier}, state, code)
	return usecase.Identity{Subject: user.Subject}, err
}
