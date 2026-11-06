package game

type Value interface{ value() }

type BoolValue struct{ Bool bool }
type IntegerValue struct{ Integer int64 }
type UnitValue struct{}

func (BoolValue) value()    {}
func (IntegerValue) value() {}
func (UnitValue) value()    {}

// TerrainValue is a cell observation copied by value. Only land can carry
// object attributes; subsequent board changes do not affect the observation.
type TerrainValue interface {
	Value
	terrain()
}

type LandTerrain struct {
	Goal       bool
	Checkpoint bool
	Luggage    bool
}

type RockTerrain struct{}
type SeaTerrain struct{}

func (LandTerrain) value()   {}
func (RockTerrain) value()   {}
func (SeaTerrain) value()    {}
func (LandTerrain) terrain() {}
func (RockTerrain) terrain() {}
func (SeaTerrain) terrain()  {}

// ValidValue rejects nil and pointer variants at protocol boundaries.
func ValidValue(value Value) bool {
	switch value.(type) {
	case BoolValue, IntegerValue, UnitValue, LandTerrain, RockTerrain, SeaTerrain:
		return true
	default:
		return false
	}
}
