package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

type userRepositoryStub struct {
	candidate  domain.User
	result     domain.User
	setSubject string
	setName    string
	setCalled  bool
	setErr     error
}

func (s *userRepositoryStub) SetNameIfAbsent(_ context.Context, subject, name string) (domain.User, error) {
	s.setCalled = true
	s.setSubject, s.setName = subject, name
	return s.result, s.setErr
}

func (s *userRepositoryStub) CreateIfAbsentByGoogleSubject(_ context.Context, candidate domain.User) (domain.User, error) {
	s.candidate = candidate
	return s.result, nil
}

func TestSetUsername(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		label, name      string
		repoErr, wantErr error
		wantCall         bool
	}{
		{"valid", "Alice_01", nil, nil, true},
		{"empty", "", nil, usecase.ErrInvalidUsername, false},
		{"invalid", "bad name", nil, usecase.ErrInvalidUsername, false},
		{"too long", strings.Repeat("a", 33), nil, usecase.ErrInvalidUsername, false},
		{"reserved self", "self", nil, usecase.ErrInvalidUsername, false},
		{"reserved std", "std", nil, usecase.ErrInvalidUsername, false},
		{"reserved mixed case", "SeLf", nil, usecase.ErrInvalidUsername, false},
		{"taken", "Alice", usecase.ErrUsernameTaken, usecase.ErrUsernameTaken, true},
		{"already set", "Alice", usecase.ErrUsernameAlreadySet, usecase.ErrUsernameAlreadySet, true},
		{"missing user", "Alice", usecase.ErrNotFound, usecase.ErrNotFound, true},
	} {
		t.Run(tt.label, func(t *testing.T) {
			t.Parallel()
			repo := &userRepositoryStub{setErr: tt.repoErr}
			_, err := usecase.NewUser(repo).SetUsername(context.Background(), "subject", tt.name)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
			if repo.setCalled != tt.wantCall {
				t.Errorf("repository called = %v, want %v", repo.setCalled, tt.wantCall)
			}
			if repo.setCalled && (repo.setSubject != "subject" || repo.setName != tt.name) {
				t.Errorf("repository got %q, %q", repo.setSubject, repo.setName)
			}
		})
	}
}

func (s *userRepositoryStub) Get(_ context.Context, _ string) (domain.User, error) {
	return s.result, nil
}

func TestResolveGoogleIdentityDoesNotAssignName(t *testing.T) {
	t.Parallel()
	name := "existing"
	repo := &userRepositoryStub{result: domain.User{GoogleSubject: "subject", Name: &name}}
	user, err := usecase.NewUser(repo).ResolveGoogleIdentity(context.Background(), "subject")
	if err != nil {
		t.Fatal(err)
	}
	if repo.candidate.GoogleSubject != "subject" || repo.candidate.Name != nil {
		t.Errorf("candidate = %+v, want subject without a name", repo.candidate)
	}
	if user.Name == nil || *user.Name != name {
		t.Errorf("existing name = %v, want %q", user.Name, name)
	}
}
