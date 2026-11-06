package game

type Value interface{ value() }

type BoolValue struct{ Bool bool }
type IntegerValue struct{ Integer int64 }
type UnitValue struct{}

func (BoolValue) value()    {}
func (IntegerValue) value() {}
func (UnitValue) value()    {}

// TerrainValue is a cell observation copied by value. Each cell is land, rock,
// or sea. Subsequent board changes do not affect the observation.
type TerrainValue interface {
	Value
	terrain()
}

// LandTerrain has one base kind and independent luggage and goal area attributes.
// GoalArea is relative to the robot that observed the cell.
type LandTerrain struct {
	Kind       LandKind
	HasLuggage bool
	GoalArea   GoalAreaRelation
}
type RockTerrain struct{}
type SeaTerrain struct{}

type GoalAreaRelation string

const (
	NoGoalArea    GoalAreaRelation = ""
	OwnGoalArea   GoalAreaRelation = "own"
	EnemyGoalArea GoalAreaRelation = "enemy"
)

// LandKind distinguishes ordinary land from tutorial goals and checkpoints.
type LandKind interface {
	landKind()
}

type PlainLand struct{}
type GoalLand struct{}
type CheckpointLand struct{ Order int64 }

func (LandTerrain) value()       {}
func (RockTerrain) value()       {}
func (SeaTerrain) value()        {}
func (LandTerrain) terrain()     {}
func (RockTerrain) terrain()     {}
func (SeaTerrain) terrain()      {}
func (PlainLand) landKind()      {}
func (GoalLand) landKind()       {}
func (CheckpointLand) landKind() {}

// ValidValue rejects nil, pointer variants, and unset land kinds at protocol boundaries.
func ValidValue(value Value) bool {
	switch value := value.(type) {
	case BoolValue, IntegerValue, UnitValue, RockTerrain, SeaTerrain:
		return true
	case LandTerrain:
		if value.GoalArea != NoGoalArea {
			_, plain := value.Kind.(PlainLand)
			return plain && (value.GoalArea == OwnGoalArea || value.GoalArea == EnemyGoalArea)
		}
		switch value.Kind.(type) {
		case PlainLand, GoalLand, CheckpointLand:
			return true
		default:
			return false
		}
	default:
		return false
	}
}
