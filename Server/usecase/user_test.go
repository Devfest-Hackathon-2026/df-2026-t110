package usecase_test

import (
	"context"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

type userRepositoryStub struct {
	candidate domain.User
	result    domain.User
}

func (s *userRepositoryStub) CreateIfAbsentByGoogleSubject(_ context.Context, candidate domain.User) (domain.User, error) {
	s.candidate = candidate
	return s.result, nil
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
