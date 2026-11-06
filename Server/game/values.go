package game

type Value interface{ value() }

type BoolValue struct{ Bool bool }
type IntegerValue struct{ Integer int64 }
type UnitValue struct{}

func (BoolValue) value()    {}
func (IntegerValue) value() {}
func (UnitValue) value()    {}

// TerrainValue is a cell observation copied by value. Each cell is land, rock,
// or sea. Land contains one LandKind. Subsequent board changes do not affect the
// observation.
type TerrainValue interface {
	Value
	terrain()
}

// LandTerrain contains exactly one non-nil, non-pointer LandKind.
type LandTerrain struct{ Kind LandKind }
type RockTerrain struct{}
type SeaTerrain struct{}

// LandKind describes one mutually exclusive kind of land, following Docs/field.md.
// PlainLand is ordinary land; goal, checkpoint, and luggage are special land.
type LandKind interface {
	landKind()
}

type PlainLand struct{}
type GoalLand struct{}
type CheckpointLand struct{ Order int64 }
type LuggageLand struct{}

func (LandTerrain) value()       {}
func (RockTerrain) value()       {}
func (SeaTerrain) value()        {}
func (LandTerrain) terrain()     {}
func (RockTerrain) terrain()     {}
func (SeaTerrain) terrain()      {}
func (PlainLand) landKind()      {}
func (GoalLand) landKind()       {}
func (CheckpointLand) landKind() {}
func (LuggageLand) landKind()    {}

// ValidValue rejects nil, pointer variants, and unset land kinds at protocol boundaries.
func ValidValue(value Value) bool {
	switch value := value.(type) {
	case BoolValue, IntegerValue, UnitValue, RockTerrain, SeaTerrain:
		return true
	case LandTerrain:
		switch value.Kind.(type) {
		case PlainLand, GoalLand, CheckpointLand, LuggageLand:
			return true
		default:
			return false
		}
	default:
		return false
	}
}
