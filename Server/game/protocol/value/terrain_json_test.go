package value_test

import (
	json "encoding/json/v2"
	"reflect"
	"testing"

	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

func TestTerrainJSONRoundTrip(t *testing.T) {
	t.Parallel()
	for _, value := range []values.TerrainValue{
		values.LandTerrain{Kind: values.PlainLand{}}, values.LandTerrain{Kind: values.GoalLand{}, HasLuggage: true}, values.LandTerrain{Kind: values.CheckpointLand{Order: 0}}, values.LandTerrain{Kind: values.PlainLand{}, HasLuggage: true, Territory: values.OwnTerritory}, values.LandTerrain{Kind: values.PlainLand{}, Territory: values.EnemyTerritory}, values.RockTerrain{}, values.SeaTerrain{},
	} {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Errorf("marshal %T: %v", value, err)
			continue
		}
		actual, err := values.ParseTerrainJSON(encoded)
		if err != nil || !reflect.DeepEqual(actual, value) {
			t.Errorf("roundtrip %s = %#v, %v; want %#v", encoded, actual, err, value)
		}
	}
}

func TestRejectsInvalidTerrainJSON(t *testing.T) {
	t.Parallel()
	for _, input := range []string{`null`, `{}`, `{"type":"invalid"}`, `{"type":"land"}`, `{"type":"land","kind":{"type":"invalid"}}`, `{"type":"land","kind":{"type":"plain","order":2}}`, `{"type":"land","kind":{"type":"checkpoint"}}`, `{"type":"land","kind":{"type":"goal"},"territory":"own"}`, `{"type":"rock","kind":{"type":"plain"}}`, `{"type":"sea","hasLuggage":true}`, `{"type":"land","kind":{"type":"plain"},"extra":true}`} {
		if value, err := values.ParseTerrainJSON([]byte(input)); err == nil {
			t.Errorf("accepted %s as %#v", input, value)
		}
	}
	if _, err := json.Marshal(values.LandTerrain{}); err == nil {
		t.Error("serialized land without a kind")
	}
}
