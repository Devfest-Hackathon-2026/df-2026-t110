package schema_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
)

func TestPicolangPrimitiveTypeNames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		valid bool
	}{
		{name: "bool", valid: true},
		{name: "integer", valid: true},
		{name: "unit", valid: true},
		{name: "terrain", valid: true},
		{name: "field", valid: false},
		{name: "rotation", valid: true},
		{name: "distance_target", valid: true},
		{name: "direction", valid: false},
		{name: "stack", valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got schema.PrimitiveType
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

func TestStackType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		data  string
		valid bool
	}{
		{name: "integer elements", data: `{"kind":"stack","elementType":"integer"}`, valid: true},
		{name: "nested stack", data: `{"kind":"stack","elementType":{"kind":"stack","elementType":"bool"}}`, valid: true},
		{name: "missing kind", data: `{"elementType":"integer"}`, valid: false},
		{name: "wrong kind", data: `{"kind":"array","elementType":"integer"}`, valid: false},
		{name: "old vector kind", data: `{"kind":"vector","elementType":"integer"}`, valid: false},
		{name: "missing element type", data: `{"kind":"stack"}`, valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got schema.StackType
			err := json.Unmarshal([]byte(tt.data), &got)
			if (err == nil) != tt.valid {
				t.Errorf("got error %v, valid = %t", err, tt.valid)
			}
			if err != nil || !tt.valid {
				return
			}
			data, err := json.Marshal(got)
			if err != nil {
				t.Error(err)
				return
			}
			var want, roundTrip any
			if err := json.Unmarshal([]byte(tt.data), &want); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &roundTrip); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(roundTrip, want) {
				t.Errorf("round trip: got %s, want %s", data, tt.data)
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

func TestDistanceTargetLiteral(t *testing.T) {
	t.Parallel()
	var got schema.Literal
	data := []byte(`{"kind":"literal","hintId":"00000000-0000-4000-8000-000000000001","type":"distance_target","value":"territory"}`)
	if err := json.Unmarshal(data, &got); err != nil {
		t.Error(err)
	}
	if got.DistanceTarget == nil || got.DistanceTarget.Value != schema.DistanceTargetTerritory {
		t.Errorf("unexpected distance target literal: %+v", got)
	}
	data = []byte(`{"kind":"literal","hintId":"00000000-0000-4000-8000-000000000001","type":"distance_target","value":"left"}`)
	if err := json.Unmarshal(data, &got); err == nil {
		t.Error("accepted an unknown distance target")
	}
}
