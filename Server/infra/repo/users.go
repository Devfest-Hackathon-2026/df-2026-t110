package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo/gen/db"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

var _ usecase.UserRepository = (*Users)(nil)

// Users stores users in PostgreSQL.
type Users struct {
	queries *db.Queries
}

func NewUsers(pool *pgxpool.Pool) *Users {
	return &Users{queries: db.New(pool)}
}

func (r *Users) CreateIfAbsentByGoogleSubject(ctx context.Context, candidate domain.User) (domain.User, error) {
	if err := r.queries.CreateUserIfAbsent(ctx, db.CreateUserIfAbsentParams{
		GoogleSubject: candidate.GoogleSubject,
		Name:          textFromPtr(candidate.Name),
	}); err != nil {
		if isUniqueViolation(err) {
			return domain.User{}, usecase.ErrUsernameTaken
		}
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}
	return r.Get(ctx, candidate.GoogleSubject)
}

func (r *Users) Get(ctx context.Context, subject string) (domain.User, error) {
	user, err := r.queries.GetUser(ctx, subject)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, usecase.ErrNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get user: %w", err)
	}
	return userFromRow(user.GoogleSubject, user.Name), nil
}

func (r *Users) SetNameIfAbsent(ctx context.Context, subject, name string) (domain.User, error) {
	user, err := r.queries.SetUserNameIfAbsent(ctx, db.SetUserNameIfAbsentParams{
		GoogleSubject: subject,
		Name:          pgtype.Text{String: name, Valid: true},
	})
	switch {
	case err == nil:
		return userFromRow(user.GoogleSubject, user.Name), nil
	case isUniqueViolation(err):
		return domain.User{}, usecase.ErrUsernameTaken
	case errors.Is(err, pgx.ErrNoRows):
		// The update matched nothing: either the user is missing or already named.
		if _, err := r.Get(ctx, subject); err != nil {
			return domain.User{}, err
		}
		return domain.User{}, usecase.ErrUsernameAlreadySet
	default:
		return domain.User{}, fmt.Errorf("set user name: %w", err)
	}
}

func userFromRow(subject string, name pgtype.Text) domain.User {
	user := domain.User{GoogleSubject: subject}
	if name.Valid {
		user.Name = &name.String
	}
	return user
}

func textFromPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}
