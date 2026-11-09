package schema_test

import (
	json "encoding/json/v2"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
)

func TestDeliverySchema(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		data   string
		object bool
		valid  bool
	}{
		{name: "delivery", data: `{"type":"deliver_luggage","territoryOwner":1,"requiredCount":3}`, valid: true},
		{name: "second team", data: `{"type":"deliver_luggage","territoryOwner":2,"requiredCount":1}`, valid: true},
		{name: "missing count", data: `{"type":"deliver_luggage","territoryOwner":1}`},
		{name: "zero count", data: `{"type":"deliver_luggage","territoryOwner":1,"requiredCount":0}`},
		{name: "negative count", data: `{"type":"deliver_luggage","territoryOwner":1,"requiredCount":-1}`},
		{name: "fractional count", data: `{"type":"deliver_luggage","territoryOwner":1,"requiredCount":1.5}`},
		{name: "missing owner", data: `{"type":"deliver_luggage","requiredCount":1}`},
		{name: "unowned destination", data: `{"type":"deliver_luggage","territoryOwner":0,"requiredCount":1}`},
		{name: "unknown team", data: `{"type":"deliver_luggage","territoryOwner":3,"requiredCount":1}`},
		{name: "territory", data: `{"type":"territory","territoryOwner":1,"position":{"x":2,"y":3}}`, object: true, valid: true},
		{name: "territory without owner", data: `{"type":"territory","position":{"x":2,"y":3}}`, object: true},
		{name: "territory without position", data: `{"type":"territory","territoryOwner":1}`, object: true},
		{name: "territory with invalid owner", data: `{"type":"territory","territoryOwner":0,"position":{"x":2,"y":3}}`, object: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var target any = &schema.ClearCondition{}
			if tt.object {
				target = &schema.FieldObject{}
			}
			err := json.Unmarshal([]byte(tt.data), target)
			if (err == nil) != tt.valid {
				t.Errorf("Unmarshal error = %v, valid = %v", err, tt.valid)
			}
			if err != nil || !tt.valid {
				return
			}
			data, err := json.Marshal(target)
			if err != nil {
				t.Error(err)
				return
			}
			var want, got any
			if err := json.Unmarshal([]byte(tt.data), &want); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("round trip = %s, want %s", data, tt.data)
			}
		})
	}
}
