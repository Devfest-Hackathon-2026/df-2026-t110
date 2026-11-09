package usecase

import (
	"context"
	json "encoding/json/v2"
	"errors"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
)

type fieldRepoStub struct {
	field schema.FieldDefinition
}

func (s fieldRepoStub) GetField(_ context.Context, id string) (schema.FieldDefinition, error) {
	if id != s.field.Id {
		return schema.FieldDefinition{}, ErrNotFound
	}
	return s.field, nil
}

func TestFieldsGetGenerationFieldEncodesRepositoryField(t *testing.T) {
	t.Parallel()

	want := schema.FieldDefinition{
		Id:         "variant-1",
		Width:      2,
		Height:     1,
		Terrain:    [][]schema.Terrain{{schema.TerrainLand, schema.TerrainLand}},
		Objects:    []schema.FieldObject{},
		Robot:      schema.FieldDefinitionRobot{Direction: schema.DirectionEast},
		DrillLimit: 1,
	}
	fields := NewFields(fieldRepoStub{field: want})

	data, err := fields.GetGenerationField(context.Background(), "subject", "variant-1")
	if err != nil {
		t.Fatal(err)
	}
	var got schema.FieldDefinition
	if err := json.Unmarshal(data, &got, json.RejectUnknownMembers(true)); err != nil {
		t.Fatalf("decode generation field %s: %v", data, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("generation field = %+v, want %+v", got, want)
	}
}

func TestFieldsGetGenerationFieldReturnsNotFound(t *testing.T) {
	t.Parallel()

	fields := NewFields(fieldRepoStub{field: schema.FieldDefinition{Id: "variant-1"}})

	_, err := fields.GetGenerationField(context.Background(), "subject", "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}
