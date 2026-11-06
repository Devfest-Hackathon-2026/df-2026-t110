package robot_event

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

var recordHintIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func validateHintID(id protocol.HintID) error {
	if !recordHintIDPattern.MatchString(string(id)) {
		return fmt.Errorf("invalid recording hint ID %q", id)
	}
	return nil
}

func marshalRecordValue(value values.RecordValue) ([]byte, error) {
	switch value := value.(type) {
	case values.BoolValue:
		return json.Marshal(value.Bool)
	case values.IntegerValue:
		return json.Marshal(value.Integer)
	case values.UnitValue:
		return []byte("null"), nil
	case values.RotationRecordValue:
		if value.Rotation != values.Left && value.Rotation != values.Right {
			return nil, fmt.Errorf("invalid recording rotation %q", value.Rotation)
		}
		return json.Marshal(value.Rotation)
	case values.LandTerrain:
		kind, err := marshalLandKind(value.Kind)
		if err != nil {
			return nil, err
		}
		return json.Marshal(struct {
			Type string          `json:"type"`
			Kind json.RawMessage `json:"kind"`
		}{Type: "land", Kind: kind})
	case values.RockTerrain:
		return []byte(`{"type":"rock"}`), nil
	case values.SeaTerrain:
		return []byte(`{"type":"sea"}`), nil
	case values.VectorRecordValue:
		return marshalVectorElements(value.Elements)
	default:
		return nil, fmt.Errorf("invalid recording value %T", value)
	}
}

func marshalLandKind(kind values.LandKind) ([]byte, error) {
	switch kind := kind.(type) {
	case values.PlainLand:
		return []byte(`{"type":"plain"}`), nil
	case values.GoalLand:
		return []byte(`{"type":"goal"}`), nil
	case values.LuggageLand:
		return []byte(`{"type":"luggage"}`), nil
	case values.CheckpointLand:
		return json.Marshal(struct {
			Type  string `json:"type"`
			Order int64  `json:"order"`
		}{Type: "checkpoint", Order: kind.Order})
	default:
		return nil, fmt.Errorf("invalid recording land kind %T", kind)
	}
}

func marshalVectorElements(elements []values.RecordValue) ([]byte, error) {
	values := make([]json.RawMessage, len(elements))
	for i, element := range elements {
		value, err := marshalRecordValue(element)
		if err != nil {
			return nil, fmt.Errorf("vector element %d: %w", i, err)
		}
		values[i] = value
	}
	return json.Marshal(values)
}

func marshalValueUpdate(update ValueUpdate) ([]byte, error) {
	switch update := update.(type) {
	case BoolUpdate:
		return marshalPrimitiveUpdate("bool", values.BoolValue{Bool: update.Value})
	case IntegerUpdate:
		return marshalPrimitiveUpdate("integer", values.IntegerValue{Integer: update.Value})
	case UnitUpdate:
		return marshalPrimitiveUpdate("unit", values.UnitValue{})
	case TerrainUpdate:
		return marshalPrimitiveUpdate("terrain", update.Value)
	case RotationUpdate:
		return marshalPrimitiveUpdate("rotation", values.RotationRecordValue{Rotation: update.Value})
	case VectorUpdate:
		operation, err := marshalVectorOperation(update.Update)
		if err != nil {
			return nil, err
		}
		return json.Marshal(struct {
			Type   string          `json:"type"`
			Update json.RawMessage `json:"update"`
		}{Type: "vector", Update: operation})
	default:
		return nil, fmt.Errorf("invalid recording update %T", update)
	}
}

func marshalPrimitiveUpdate(kind string, value values.RecordValue) ([]byte, error) {
	encoded, err := marshalRecordValue(value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Type  string          `json:"type"`
		Value json.RawMessage `json:"value"`
	}{Type: kind, Value: encoded})
}

func marshalVectorOperation(operation VectorOperation) ([]byte, error) {
	var name string
	var value []byte
	var err error
	switch operation := operation.(type) {
	case VectorReplace:
		name = "replace"
		value, err = marshalVectorElements(operation.Value)
	case PushBack:
		name = "push_back"
		value, err = marshalRecordValue(operation.Value)
	case PopBack:
		return []byte(`{"op":"pop_back"}`), nil
	default:
		return nil, fmt.Errorf("invalid recording vector operation %T", operation)
	}
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Op    string          `json:"op"`
		Value json.RawMessage `json:"value"`
	}{Op: name, Value: value})
}
