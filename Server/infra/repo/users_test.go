package repo

import (
	"context"
	"errors"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

func TestUsersCreateIfAbsentByGoogleSubject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		existing *domain.User
		want     domain.User
	}{
		{
			name: "creates an unnamed user",
			want: domain.User{GoogleSubject: "sub"},
		},
		{
			name:     "keeps an existing user unchanged",
			existing: &domain.User{GoogleSubject: "sub", Name: ptr("alice")},
			want:     domain.User{GoogleSubject: "sub", Name: ptr("alice")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pool := newTestPool(t)
			if tt.existing != nil {
				mustCreateUser(t, pool, tt.existing.GoogleSubject, tt.existing.Name)
			}

			got, err := NewUsers(pool).CreateIfAbsentByGoogleSubject(context.Background(), domain.User{GoogleSubject: "sub"})
			if err != nil {
				t.Fatalf("CreateIfAbsentByGoogleSubject() error = %v", err)
			}
			if got.GoogleSubject != tt.want.GoogleSubject {
				t.Errorf("GoogleSubject = %q, want %q", got.GoogleSubject, tt.want.GoogleSubject)
			}
			if describeName(got.Name) != describeName(tt.want.Name) {
				t.Errorf("Name = %s, want %s", describeName(got.Name), describeName(tt.want.Name))
			}
		})
	}
}

func TestUsersGetMissing(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)

	_, err := NewUsers(pool).Get(context.Background(), "missing")
	if !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("Get() error = %v, want %v", err, usecase.ErrNotFound)
	}
}

func TestUsersSetNameIfAbsent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		users   []domain.User
		wantErr error
	}{
		{
			name:  "sets the name of an unnamed user",
			users: []domain.User{{GoogleSubject: "sub"}},
		},
		{
			name:    "rejects a missing user",
			wantErr: usecase.ErrNotFound,
		},
		{
			name:    "rejects a user who already has a name",
			users:   []domain.User{{GoogleSubject: "sub", Name: ptr("bob")}},
			wantErr: usecase.ErrUsernameAlreadySet,
		},
		{
			name:    "rejects a name held by another user",
			users:   []domain.User{{GoogleSubject: "sub"}, {GoogleSubject: "other", Name: ptr("alice")}},
			wantErr: usecase.ErrUsernameTaken,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pool := newTestPool(t)
			for _, u := range tt.users {
				mustCreateUser(t, pool, u.GoogleSubject, u.Name)
			}

			got, err := NewUsers(pool).SetNameIfAbsent(context.Background(), "sub", "alice")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("SetNameIfAbsent() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got.Name == nil || *got.Name != "alice" {
				t.Errorf("Name = %s, want %q", describeName(got.Name), "alice")
			}
		})
	}
}
