package value

import (
	json "encoding/json/v2"
	"fmt"
)

type landKindJSON struct {
	Type  string `json:"type"`
	Order *int64 `json:"order,omitempty"`
}
type terrainJSON struct {
	Type       string            `json:"type"`
	Kind       *landKindJSON     `json:"kind,omitempty"`
	HasLuggage bool              `json:"hasLuggage,omitzero"`
	Territory  TerritoryRelation `json:"territory,omitzero"`
}

func (v LandTerrain) MarshalJSON() ([]byte, error) {
	if !ValidValue(v) {
		return nil, fmt.Errorf("invalid land terrain")
	}
	kind := &landKindJSON{}
	switch k := v.Kind.(type) {
	case PlainLand:
		kind.Type = "plain"
	case GoalLand:
		kind.Type = "goal"
	case CheckpointLand:
		kind.Type = "checkpoint"
		kind.Order = &k.Order
	}
	return json.Marshal(terrainJSON{Type: "land", Kind: kind, HasLuggage: v.HasLuggage, Territory: v.Territory})
}
func (RockTerrain) MarshalJSON() ([]byte, error) { return []byte(`{"type":"rock"}`), nil }
func (SeaTerrain) MarshalJSON() ([]byte, error)  { return []byte(`{"type":"sea"}`), nil }

// ParseTerrainJSON reads a complete terrain observation for the interactive CLI.
func ParseTerrainJSON(data []byte) (TerrainValue, error) {
	var input terrainJSON
	if err := json.Unmarshal(data, &input, json.RejectUnknownMembers(true)); err != nil {
		return nil, err
	}
	if input.Type == "land" && input.Kind != nil {
		var kind LandKind
		switch input.Kind.Type {
		case "plain":
			kind = PlainLand{}
		case "goal":
			kind = GoalLand{}
		case "checkpoint":
			if input.Kind.Order == nil {
				return nil, fmt.Errorf("checkpoint requires order")
			}
			kind = CheckpointLand{Order: *input.Kind.Order}
		}
		if input.Kind.Type != "checkpoint" && input.Kind.Order != nil {
			return nil, fmt.Errorf("only checkpoint has order")
		}
		result := LandTerrain{Kind: kind, HasLuggage: input.HasLuggage, Territory: input.Territory}
		if ValidValue(result) {
			return result, nil
		}
	} else if input.Kind == nil && !input.HasLuggage && input.Territory == NoTerritory {
		switch input.Type {
		case "rock":
			return RockTerrain{}, nil
		case "sea":
			return SeaTerrain{}, nil
		}
	}
	return nil, fmt.Errorf("invalid terrain observation")
}
