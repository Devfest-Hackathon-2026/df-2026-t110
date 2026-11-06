package memory

import (
	"reflect"

	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

// Each map entry contains an ID and an entry pointer; the entry contains the
// value interface and reference count. Concrete payloads are counted separately.
var entryOverhead = uint64(reflect.TypeFor[ID]().Size() +
	reflect.TypeFor[*entry]().Size() + reflect.TypeFor[entry]().Size())

// entryBytes only receives validated, immutable values. List edges count as IDs
// here; their targets are accounted for by their own entries.
func entryBytes(value Value) uint64 {
	size := entryOverhead + uint64(reflect.TypeOf(value).Size())
	switch v := value.(type) {
	case Terrain:
		size += uint64(reflect.TypeOf(v.Value).Size())
		if land, ok := v.Value.(values.LandTerrain); ok {
			size += uint64(reflect.TypeOf(land.Kind).Size())
		}
	case Rotation:
		size += uint64(len(v.Value))
	}
	return size
}
