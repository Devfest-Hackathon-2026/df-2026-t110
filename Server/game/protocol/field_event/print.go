package field_event

import (
	json "encoding/json/v2"

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
	type payload PrintTerrainResult
	return json.Marshal(struct {
		Kind string `json:"kind"`
		payload
	}{Kind: "print_terrain", payload: payload(r)})
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
