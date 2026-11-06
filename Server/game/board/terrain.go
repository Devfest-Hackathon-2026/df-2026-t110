package board

// Terrain describes a cell of the shared board.
type Terrain interface{ terrain() }

type TeamID int

const (
	TeamID1 TeamID = iota + 1
	TeamID2
)

type LandTerrain struct {
	Kind       LandKind
	HasLuggage bool
	// TerritoryOwner is 0 outside a territory.
	TerritoryOwner TeamID
	Terrain
}

type RockTerrain struct{ Terrain }
type SeaTerrain struct{ Terrain }

type LandKind interface{ landKind() }

type PlainLand struct{ LandKind }
type GoalLand struct{ LandKind }
type CheckpointLand struct {
	Order int64
	LandKind
}

func validTerrain(terrain Terrain) bool {
	switch terrain := terrain.(type) {
	case LandTerrain:
		switch terrain.Kind.(type) {
		case PlainLand:
			return true
		case GoalLand, CheckpointLand:
			return terrain.TerritoryOwner == 0
		}
	case RockTerrain, SeaTerrain:
		return true
	}
	return false
}
