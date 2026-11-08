package googleauth

import (
	"context"

	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

var _ usecase.IdentityProvider = Provider{}

// Provider adapts Client to usecase.IdentityProvider.
type Provider struct {
	Client *Client
}

func (p Provider) Begin() (usecase.AuthorizationRequest, error) {
	a, err := p.Client.Begin()
	if err != nil {
		return usecase.AuthorizationRequest{}, err
	}
	return usecase.AuthorizationRequest{URL: a.URL, State: a.State, Nonce: a.Nonce, CodeVerifier: a.CodeVerifier}, nil
}

func (p Provider) Complete(ctx context.Context, request usecase.AuthorizationRequest, state, code string) (usecase.Identity, error) {
	user, err := p.Client.Complete(ctx, Authorization{URL: request.URL, State: request.State, Nonce: request.Nonce, CodeVerifier: request.CodeVerifier}, state, code)
	if err != nil {
		return usecase.Identity{}, err
	}
	return usecase.Identity{Subject: user.Subject}, nil
}
