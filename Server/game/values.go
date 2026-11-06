package game

type Value interface{ value() }

type BoolValue struct{ Bool bool }
type IntegerValue struct{ Integer int64 }
type UnitValue struct{}

func (BoolValue) value()    {}
func (IntegerValue) value() {}
func (UnitValue) value()    {}

// TerrainValue is a cell observation copied by value. Each cell is exactly one
// of plain land, goal, checkpoint, luggage, rock, or sea. Subsequent board changes
// do not affect the observation.
type TerrainValue interface {
	Value
	terrain()
}

// LandTerrainValue covers the four mutually exclusive walkable terrain kinds.
type LandTerrainValue interface {
	TerrainValue
	land()
}

type LandTerrain struct{}
type GoalTerrain struct{}
type CheckpointTerrain struct{}
type LuggageTerrain struct{}
type RockTerrain struct{}
type SeaTerrain struct{}

func (LandTerrain) value()         {}
func (GoalTerrain) value()         {}
func (CheckpointTerrain) value()   {}
func (LuggageTerrain) value()      {}
func (RockTerrain) value()         {}
func (SeaTerrain) value()          {}
func (LandTerrain) terrain()       {}
func (GoalTerrain) terrain()       {}
func (CheckpointTerrain) terrain() {}
func (LuggageTerrain) terrain()    {}
func (RockTerrain) terrain()       {}
func (SeaTerrain) terrain()        {}
func (LandTerrain) land()          {}
func (GoalTerrain) land()          {}
func (CheckpointTerrain) land()    {}
func (LuggageTerrain) land()       {}

// ValidValue rejects nil and pointer variants at protocol boundaries.
func ValidValue(value Value) bool {
	switch value.(type) {
	case BoolValue, IntegerValue, UnitValue, LandTerrain, GoalTerrain,
		CheckpointTerrain, LuggageTerrain, RockTerrain, SeaTerrain:
		return true
	default:
		return false
	}
}
