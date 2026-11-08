package repo

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"gopkg.in/yaml.v3"
)

var _ usecase.TutorialRepository = (*Textbooks)(nil)

// Textbooks keeps validated pages indexed by ID after loading the source.
type Textbooks struct {
	byID map[string]domain.Tutorial
	list []domain.Tutorial
}

// NewTextbooks validates and indexes all pages at startup.
func NewTextbooks(ctx context.Context, pages fs.FS) (*Textbooks, error) {
	byID, list, err := loadTextbooks(ctx, pages)
	if err != nil {
		return nil, err
	}
	return &Textbooks{byID: byID, list: list}, nil
}

func loadTextbooks(ctx context.Context, pages fs.FS) (map[string]domain.Tutorial, []domain.Tutorial, error) {
	byID := make(map[string]domain.Tutorial)
	var list []domain.Tutorial
	err := fs.WalkDir(pages, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk textbook %s: %w", path, walkErr)
		}
		if entry.IsDir() || (!strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml")) {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		raw, err := fs.ReadFile(pages, path)
		if err != nil {
			return fmt.Errorf("read textbook %s: %w", path, err)
		}
		tutorial, err := parseTextbook(path, raw)
		if err != nil {
			return err
		}
		if _, exists := byID[tutorial.ID]; exists {
			return fmt.Errorf("duplicate textbook id %q", tutorial.ID)
		}
		byID[tutorial.ID] = tutorial
		list = append(list, tutorial)
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("load textbooks: %w", err)
	}
	slices.SortFunc(list, func(a, b domain.Tutorial) int { return strings.Compare(a.ID, b.ID) })
	return byID, list, nil
}

func (r *Textbooks) List(ctx context.Context) ([]domain.Tutorial, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return slices.Clone(r.list), nil
}

func (r *Textbooks) Get(ctx context.Context, id string) (domain.Tutorial, error) {
	if err := ctx.Err(); err != nil {
		return domain.Tutorial{}, err
	}
	tutorial, ok := r.byID[id]
	if !ok {
		return domain.Tutorial{}, usecase.ErrNotFound
	}
	return tutorial, nil
}

func parseTextbook(path string, raw []byte) (domain.Tutorial, error) {
	var page map[string]any
	if err := yaml.Unmarshal(raw, &page); err != nil {
		return domain.Tutorial{}, fmt.Errorf("parse textbook %s: %w", path, err)
	}
	data, err := json.Marshal(page)
	if err != nil {
		return domain.Tutorial{}, fmt.Errorf("encode textbook %s: %w", path, err)
	}
	var detail schema.TutorialDetail
	if err := json.Unmarshal(data, &detail); err != nil {
		return domain.Tutorial{}, fmt.Errorf("validate textbook %s: %w", path, err)
	}
	var initialProgram jsontext.Value
	if value, ok := page["initialProgram"]; ok {
		initialProgram, err = json.Marshal(value)
		if err != nil {
			return domain.Tutorial{}, fmt.Errorf("encode initial program %s: %w", path, err)
		}
	}
	return domain.Tutorial{
		ID: detail.Id, Title: detail.Title, Description: detail.Description,
		Stage: detail.Stage, Goal: detail.Goal, Lesson: detail.Lesson,
		Guide: detail.Guide, InitialProgram: initialProgram,
	}, nil
}
