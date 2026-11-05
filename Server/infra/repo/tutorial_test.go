package repo

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

// readTutorial decodes a testdata fixture; decoding the stage validates it
// against the generated schema types.
func readTutorial(t *testing.T, id string) domain.Tutorial {
	t.Helper()
	raw, err := fs.ReadFile(os.DirFS("testdata"), id+".json")
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Title       string                 `json:"title"`
		Description string                 `json:"description"`
		Stage       schema.StageDefinition `json:"stage"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	return domain.Tutorial{ID: id, Title: file.Title, Description: file.Description, Stage: file.Stage}
}

func TestTutorials(t *testing.T) {
	t.Parallel()
	fixture := readTutorial(t, "tutorial-1")
	r := NewTutorials(domain.Tutorial{ID: "tutorial-2", Title: "Second"}, fixture)

	list, err := r.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != "tutorial-1" || list[1].ID != "tutorial-2" {
		t.Errorf("List() = %+v, want tutorial-1, tutorial-2 in ID order", list)
	}
	got, err := r.Get(context.Background(), "tutorial-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != fixture.Title || got.Description != fixture.Description || len(got.Stage.Variants) != len(fixture.Stage.Variants) {
		t.Errorf("Get(tutorial-1) = %+v, want %+v", got, fixture)
	}
	if _, err := r.Get(context.Background(), "missing"); !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("Get(missing) error = %v, want ErrNotFound", err)
	}
}
