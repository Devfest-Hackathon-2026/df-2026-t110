package engine

// Terrain describes a cell of the shared board.
type Terrain interface{ terrain() }

type TeamID int

const (
	TeamID1 TeamID = iota + 1
	TeamID2
)

type LandTerrain struct {
	kind           LandKind
	hasLuggage     bool
	territoryOwner TeamID
}

func NewLandTerrain(kind LandKind, hasLuggage bool, territoryOwner TeamID) LandTerrain {
	return LandTerrain{kind: kind, hasLuggage: hasLuggage, territoryOwner: territoryOwner}
}

func (t LandTerrain) Kind() LandKind   { return t.kind }
func (t LandTerrain) HasLuggage() bool { return t.hasLuggage }

// TerritoryOwner returns zero outside a territory.
func (t LandTerrain) TerritoryOwner() TeamID { return t.territoryOwner }

type RockTerrain struct{}
type SeaTerrain struct{}

func (LandTerrain) terrain() {}
func (RockTerrain) terrain() {}
func (SeaTerrain) terrain()  {}

type LandKind interface{ landKind() }

type PlainLand struct{}
type GoalLand struct{}
type CheckpointLand struct{ order int64 }

func NewCheckpointLand(order int64) CheckpointLand { return CheckpointLand{order: order} }

func (c CheckpointLand) Order() int64 { return c.order }

func (PlainLand) landKind()      {}
func (GoalLand) landKind()       {}
func (CheckpointLand) landKind() {}

func validTerrain(terrain Terrain) bool {
	switch terrain := terrain.(type) {
	case LandTerrain:
		switch terrain.kind.(type) {
		case PlainLand:
			return true
		case GoalLand, CheckpointLand:
			return terrain.territoryOwner == 0
		}
	case RockTerrain, SeaTerrain:
		return true
	}
	return false
}
