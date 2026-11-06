package usecase

import (
	"context"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

// UserRepository stores users and enforces unique subjects and assigned names.
type UserRepository interface {
	// Creation is atomic and retains existing users unchanged.
	CreateIfAbsentByGoogleSubject(ctx context.Context, candidate domain.User) (domain.User, error)
	Get(ctx context.Context, subject string) (domain.User, error)
}

type User struct{ repo UserRepository }

func NewUser(repo UserRepository) *User { return &User{repo: repo} }

// ResolveGoogleIdentity returns the user for a verified Google identity, creating one if needed.
func (u *User) ResolveGoogleIdentity(ctx context.Context, subject string) (domain.User, error) {
	if subject == "" {
		return domain.User{}, fmt.Errorf("%w: missing subject", ErrInvalidLogin)
	}
	return u.repo.CreateIfAbsentByGoogleSubject(ctx, domain.User{GoogleSubject: subject})
}

func (u *User) Get(ctx context.Context, subject string) (domain.User, error) {
	return u.repo.Get(ctx, subject)
}
