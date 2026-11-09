package usecase

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
)

// FieldRepository retrieves published tutorial variants by field ID.
type FieldRepository interface {
	GetField(ctx context.Context, id string) (schema.FieldDefinition, error)
}

type Fields struct{ repo FieldRepository }

func NewFields(repo FieldRepository) *Fields { return &Fields{repo: repo} }

func (f *Fields) Get(ctx context.Context, id string) (schema.FieldDefinition, error) {
	return f.repo.GetField(ctx, id)
}

// GetGenerationField uses the same field IDs and source as the field API.
// Battle layouts cannot be represented by the single-robot Field schema.
func (f *Fields) GetGenerationField(ctx context.Context, _ string, id string) (jsontext.Value, error) {
	field, err := f.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(field)
	if err != nil {
		return nil, fmt.Errorf("encode generation field %q: %w", id, err)
	}
	return jsontext.Value(data), nil
}
