package schema_test

import (
	"encoding/json"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
)

func TestPicolangTypeNames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		valid bool
	}{
		{name: "terrain", valid: false},
		{name: "field", valid: true},
		{name: "rotation", valid: true},
		{name: "direction", valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got schema.Type
			err := json.Unmarshal([]byte(`"`+tt.name+`"`), &got)
			if (err == nil) != tt.valid {
				t.Errorf("type %q: got error %v, valid = %t", tt.name, err, tt.valid)
			}
			if err == nil && string(got) != tt.name {
				t.Errorf("got type %q, want %q", got, tt.name)
			}
		})
	}
}

func TestCardinalDirections(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		valid bool
	}{
		{name: "north", valid: true},
		{name: "south", valid: true},
		{name: "west", valid: true},
		{name: "east", valid: true},
		{name: "up", valid: false},
		{name: "down", valid: false},
		{name: "left", valid: false},
		{name: "right", valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got schema.Direction
			err := json.Unmarshal([]byte(`"`+tt.name+`"`), &got)
			if (err == nil) != tt.valid {
				t.Errorf("direction %q: got error %v, valid = %t", tt.name, err, tt.valid)
			}
		})
	}
}

func TestRotationLiteral(t *testing.T) {
	t.Parallel()
	data := []byte(`{"kind":"literal","hintId":"00000000-0000-4000-8000-000000000001","type":"rotation","value":"left"}`)
	var got schema.Literal
	if err := json.Unmarshal(data, &got); err != nil {
		t.Error(err)
	}
	if got.Rotation == nil || got.Rotation.Value != schema.RotationLiteralValueLeft {
		t.Errorf("unexpected rotation literal: %+v", got)
	}
}
