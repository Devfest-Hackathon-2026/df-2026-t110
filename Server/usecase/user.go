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
	// SetNameIfAbsent atomically assigns name only when the user exists, has no
	// name, and name is unique. It returns ErrNotFound, ErrUsernameAlreadySet,
	// or ErrUsernameTaken when the corresponding condition fails.
	SetNameIfAbsent(ctx context.Context, subject, name string) (domain.User, error)
}

type User struct{ repo UserRepository }

func NewUser(repo UserRepository) *User { return &User{repo: repo} }

// ResolveIdentity returns the user for a verified provider identity, creating one if needed.
func (u *User) ResolveIdentity(ctx context.Context, subject string) (domain.User, error) {
	if subject == "" {
		return domain.User{}, fmt.Errorf("%w: missing subject", ErrInvalidLogin)
	}
	return u.repo.CreateIfAbsentByGoogleSubject(ctx, domain.User{GoogleSubject: subject})
}

func (u *User) Get(ctx context.Context, subject string) (domain.User, error) {
	return u.repo.Get(ctx, subject)
}

// SetUsername assigns a name once. The caller must authorize the subject.
func (u *User) SetUsername(ctx context.Context, subject, name string) (domain.User, error) {
	if !domain.IsValidUsername(name) {
		return domain.User{}, ErrInvalidUsername
	}
	return u.repo.SetNameIfAbsent(ctx, subject, name)
}
