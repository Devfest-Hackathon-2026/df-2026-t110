package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

func TestTutorials(t *testing.T) {
	t.Parallel()
	tutorials := usecase.NewTutorials(repo.NewTutorials(
		domain.Tutorial{ID: "b", Title: "B"},
		domain.Tutorial{ID: "a", Title: "A"},
	))

	list, err := tutorials.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != "a" || list[1].ID != "b" {
		t.Errorf("List() = %+v, want tutorials a, b in ID order", list)
	}
	if got, err := tutorials.Get(context.Background(), "b"); err != nil || got.Title != "B" {
		t.Errorf("Get(b) = %+v, %v", got, err)
	}
	if _, err := tutorials.Get(context.Background(), "missing"); !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("Get(missing) error = %v, want ErrNotFound", err)
	}
}
