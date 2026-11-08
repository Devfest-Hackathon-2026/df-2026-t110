package repo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

const textbookPage = `id: first-step
title: First step
description: Move to the goal
goal: Reach the goal
guide: [Move forward]
stage:
  variants:
    - id: first-step-1
      width: 2
      height: 1
      terrain: [[land, land]]
      objects:
        - type: goal
          position: {x: 1, y: 0}
      robot:
        position: {x: 0, y: 0}
        direction: east
      drillLimit: 0
  clearCondition:
    type: reach_goal
`

func TestTextbooksLoadAndIndex(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "01.yaml")
	if err := os.WriteFile(path, []byte(textbookPage), 0600); err != nil {
		t.Fatal(err)
	}
	repo, err := NewTextbooks(context.Background(), os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}
	list, err := repo.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "first-step" || list[0].Title != "First step" {
		t.Errorf("List() = %+v, want first-step", list)
	}
	got, err := repo.Get(context.Background(), "first-step")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Stage.Variants) != 1 || got.Goal == nil || *got.Goal != "Reach the goal" {
		t.Errorf("Get(first-step) = %+v, want stage and goal", got)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(textbookPage, "First step", "Updated title", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	unchanged, err := repo.Get(context.Background(), "first-step")
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Title != "First step" {
		t.Errorf("loaded page changed after startup: %q", unchanged.Title)
	}
	reloaded, err := NewTextbooks(context.Background(), os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}
	updated, err := reloaded.Get(context.Background(), "first-step")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Updated title" {
		t.Errorf("Get after restart: title = %q, want Updated title", updated.Title)
	}
	if err := os.WriteFile(path, []byte("stage: [invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewTextbooks(context.Background(), os.DirFS(dir)); err == nil {
		t.Error("NewTextbooks() succeeded with invalid YAML")
	}
	if _, err := repo.Get(context.Background(), "missing"); !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("Get(missing) error = %v, want ErrNotFound", err)
	}
}
