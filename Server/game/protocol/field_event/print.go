package field_event

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

type PrintUnitResult struct {
	RobotID      protocol.RobotID `json:"robotId"`
	ActionResult `json:"-"`
}

func (r PrintUnitResult) MarshalJSON() ([]byte, error) {
	type payload PrintUnitResult
	return json.Marshal(struct {
		Kind string `json:"kind"`
		payload
	}{Kind: "print_unit", payload: payload(r)})
}

type PrintTerrainResult struct {
	RobotID      protocol.RobotID    `json:"robotId"`
	Terrain      values.TerrainValue `json:"terrain"`
	ActionResult `json:"-"`
}

func (r PrintTerrainResult) MarshalJSON() ([]byte, error) {
	terrain, err := marshalPrintedTerrain(r.Terrain)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Kind    string           `json:"kind"`
		RobotID protocol.RobotID `json:"robotId"`
		Terrain jsontext.Value   `json:"terrain"`
	}{Kind: "print_terrain", RobotID: r.RobotID, Terrain: terrain})
}

func marshalPrintedTerrain(terrain values.TerrainValue) ([]byte, error) {
	switch terrain := terrain.(type) {
	case values.LandTerrain:
		if !values.ValidValue(terrain) {
			return nil, fmt.Errorf("invalid print terrain: %T", terrain)
		}
		type landKind struct {
			Type  string `json:"type"`
			Order *int64 `json:"order,omitempty"`
		}
		kind := landKind{}
		switch value := terrain.Kind.(type) {
		case values.PlainLand:
			kind.Type = "plain"
		case values.GoalLand:
			kind.Type = "goal"
		case values.CheckpointLand:
			kind.Type, kind.Order = "checkpoint", &value.Order
		}
		return json.Marshal(struct {
			Type       string                   `json:"type"`
			Kind       landKind                 `json:"kind"`
			HasLuggage bool                     `json:"hasLuggage,omitzero"`
			Territory  values.TerritoryRelation `json:"territory,omitzero"`
		}{Type: "land", Kind: kind, HasLuggage: terrain.HasLuggage, Territory: terrain.Territory})
	case values.RockTerrain:
		return []byte(`{"type":"rock"}`), nil
	case values.SeaTerrain:
		return []byte(`{"type":"sea"}`), nil
	default:
		return nil, fmt.Errorf("invalid print terrain: %T", terrain)
	}
}

type PrintRotationResult struct {
	RobotID      protocol.RobotID `json:"robotId"`
	Rotation     values.Rotation  `json:"rotation"`
	ActionResult `json:"-"`
}

func (r PrintRotationResult) MarshalJSON() ([]byte, error) {
	type payload PrintRotationResult
	return json.Marshal(struct {
		Kind string `json:"kind"`
		payload
	}{Kind: "print_rotation", payload: payload(r)})
}

type PrintDistanceTargetResult struct {
	RobotID        protocol.RobotID      `json:"robotId"`
	DistanceTarget values.DistanceTarget `json:"distanceTarget"`
	ActionResult   `json:"-"`
}

func (r PrintDistanceTargetResult) MarshalJSON() ([]byte, error) {
	type payload PrintDistanceTargetResult
	return json.Marshal(struct {
		Kind string `json:"kind"`
		payload
	}{Kind: "print_distance_target", payload: payload(r)})
}
