package repo

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"io/fs"
	"strings"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"gopkg.in/yaml.v3"
)

var _ usecase.TutorialRepository = (*Textbooks)(nil)

// Textbooks reads the current YAML files on each request.
type Textbooks struct{ pages fs.FS }

func NewTextbooks(pages fs.FS) *Textbooks { return &Textbooks{pages: pages} }

func (r *Textbooks) List(ctx context.Context) ([]domain.Tutorial, error) {
	tutorials, err := r.readAll(ctx)
	if err != nil {
		return nil, err
	}
	return tutorials, nil
}

func (r *Textbooks) Get(ctx context.Context, id string) (domain.Tutorial, error) {
	tutorials, err := r.readAll(ctx)
	if err != nil {
		return domain.Tutorial{}, err
	}
	for _, tutorial := range tutorials {
		if tutorial.ID == id {
			return tutorial, nil
		}
	}
	return domain.Tutorial{}, usecase.ErrNotFound
}

func (r *Textbooks) readAll(ctx context.Context) ([]domain.Tutorial, error) {
	var tutorials []domain.Tutorial
	seen := make(map[string]struct{})
	err := fs.WalkDir(r.pages, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || (!strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml")) {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		raw, err := fs.ReadFile(r.pages, path)
		if err != nil {
			return err
		}
		var page map[string]any
		if err := yaml.Unmarshal(raw, &page); err != nil {
			return fmt.Errorf("parse textbook %s: %w", path, err)
		}
		data, err := json.Marshal(page)
		if err != nil {
			return fmt.Errorf("encode textbook %s: %w", path, err)
		}
		var detail schema.TutorialDetail
		if err := json.Unmarshal(data, &detail); err != nil {
			return fmt.Errorf("validate textbook %s: %w", path, err)
		}
		if _, exists := seen[detail.Id]; exists {
			return fmt.Errorf("duplicate textbook id %q", detail.Id)
		}
		seen[detail.Id] = struct{}{}
		var initialProgram jsontext.Value
		if value, ok := page["initialProgram"]; ok {
			initialProgram, err = json.Marshal(value)
			if err != nil {
				return fmt.Errorf("encode initial program %s: %w", path, err)
			}
		}
		tutorials = append(tutorials, domain.Tutorial{
			ID: detail.Id, Title: detail.Title, Description: detail.Description,
			Stage: detail.Stage, Goal: detail.Goal, Lesson: detail.Lesson,
			Guide: detail.Guide, InitialProgram: initialProgram,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tutorials, nil
}
