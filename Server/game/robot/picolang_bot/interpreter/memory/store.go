package memory

import (
	"fmt"
	"math"

	robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

type entry struct {
	value      Value
	references uint64
}

// Store owns values and their reference counts. Its zero value is ready to use.
// It must not be copied after first use and is not safe for concurrent use.
// Value payloads never change; reference counts are separate bookkeeping.
// The common root always has one reference owned by the store itself.
type Store struct {
	events      EventSink
	entries     map[ID]*entry
	nextID      ID
	memoryUsage uint64
	peakUsage   uint64
}

// EventSink receives successful public operations in order. Implicit child-edge
// retains and recursive releases are part of their initiating operation.
type EventSink interface {
	Add(robotevent.PicolangEvent)
}

// New enables operation recording before the store is first used. A nil sink
// disables recording; the zero Store also remains ready for unlogged execution.
func New(events EventSink) *Store { return &Store{events: events} }

// Add stores a copy of value with one owned reference for the caller. List nodes
// and some retain their outgoing references. Repeated roots share RootID and acquire
// another reference. On error, storage and reference counts are unchanged.
func (s *Store) Add(value Value) (ID, error) {
	s.initialize()
	switch v := value.(type) {
	case Bool, Integer, Unit:
	case None:
		value = None{}
	case Some:
		if err := s.canRetain(v.Element, 1); err != nil {
			return 0, err
		}
		value = Some{Element: v.Element}
	case Terrain:
		// ValidValue also rejects pointers nested inside land observations.
		if !values.ValidValue(v.Value) {
			return 0, fmt.Errorf("invalid terrain value %T", v.Value)
		}
		// Terrain variants embed an interface to provide their marker methods.
		// Preserve only observation data, never a reference hidden in that marker.
		switch terrain := v.Value.(type) {
		case values.LandTerrain:
			v.Value = values.LandTerrain{Kind: terrain.Kind, HasLuggage: terrain.HasLuggage, Territory: terrain.Territory}
		case values.RockTerrain:
			v.Value = values.RockTerrain{}
		case values.SeaTerrain:
			v.Value = values.SeaTerrain{}
		}
		value = v
	case Rotation:
		if v.Value != values.Left && v.Value != values.Right {
			return 0, fmt.Errorf("invalid rotation %q", v.Value)
		}
	case DistanceTarget:
		if !v.Value.Valid() {
			return 0, fmt.Errorf("invalid distance target %q", v.Value)
		}
	case ListRoot:
		return RootID, s.Retain(RootID)
	case ListNode:
		prev, exists := s.entries[v.Prev]
		if !exists {
			return 0, fmt.Errorf("unknown previous list ID %d", v.Prev)
		}
		switch prev.value.(type) {
		case ListRoot, ListNode:
		default:
			return 0, fmt.Errorf("previous ID %d is not a list", v.Prev)
		}
		if err := s.canRetain(v.Prev, 1); err != nil {
			return 0, err
		}
		count := uint64(1)
		if v.Prev == v.Value {
			count++
		}
		if err := s.canRetain(v.Value, count); err != nil {
			return 0, err
		}
	default:
		return 0, fmt.Errorf("unsupported memory value %T", value)
	}
	if s.nextID == math.MaxUint64 {
		return 0, fmt.Errorf("memory value IDs exhausted")
	}
	size := entryBytes(value)
	if size > math.MaxUint64-s.memoryUsage {
		return 0, fmt.Errorf("memory usage overflow")
	}
	if node, ok := value.(ListNode); ok {
		s.entries[node.Prev].references++
		s.entries[node.Value].references++
	}
	if some, ok := value.(Some); ok {
		s.entries[some.Element].references++
	}
	s.nextID++
	s.insert(s.nextID, value, size)
	if s.events != nil {
		s.events.Add(robotevent.MemoryAllocateEvent{ID: s.nextID, Value: allocationValue(value)})
	}
	return s.nextID, nil
}

func (s *Store) initialize() {
	if s.entries == nil {
		s.entries = make(map[ID]*entry)
		s.insert(RootID, ListRoot{}, entryBytes(ListRoot{}))
	}
}

func (s *Store) insert(id ID, value Value, size uint64) {
	s.entries[id] = &entry{value: value, references: 1}
	s.memoryUsage += size
	s.peakUsage = max(s.peakUsage, s.memoryUsage)
}

// MemoryUsage returns an O(1) estimate in bytes of the live entries, including
// the common root, IDs, entry pointers, reference counts, and value payloads.
// Shared IDs are counted once. Store itself, map capacity, and Go allocator
// overhead are excluded. Usage is updated on insertion and final release.
func (s *Store) MemoryUsage() uint64 {
	s.initialize()
	return s.memoryUsage
}

// PeakMemoryUsage includes temporary values released within a VM transition so
// a resource limit cannot miss allocations between its checks.
func (s *Store) PeakMemoryUsage() uint64 {
	s.initialize()
	return s.peakUsage
}

// Get returns a copy of the payload without acquiring a reference. The caller
// must own a reference for as long as it uses the ID or any child IDs.
func (s *Store) Get(id ID) (Value, bool) {
	s.initialize()
	e, exists := s.entries[id]
	if !exists {
		return nil, false
	}
	return e.value, true
}

// RefCount includes caller-owned references, incoming compound-value edges, and the
// store-owned reference to RootID.
func (s *Store) RefCount(id ID) (uint64, bool) {
	s.initialize()
	e, exists := s.entries[id]
	if !exists {
		return 0, false
	}
	return e.references, true
}

// Retain acquires another owned reference to an existing value.
func (s *Store) Retain(id ID) error {
	s.initialize()
	if err := s.canRetain(id, 1); err != nil {
		return err
	}
	s.entries[id].references++
	if s.events != nil {
		s.events.Add(robotevent.MemoryRetainEvent{ID: id})
	}
	return nil
}

func (s *Store) canRetain(id ID, count uint64) error {
	e, exists := s.entries[id]
	if !exists {
		return fmt.Errorf("unknown memory value ID %d", id)
	}
	if count > math.MaxUint64-e.references {
		return fmt.Errorf("reference count overflow for memory value ID %d", id)
	}
	return nil
}

// Release drops one owned reference. Values reaching zero are removed, releasing
// their outgoing references as well. Traversal is iterative for long lists.
// IDs are never reused. The store-owned root reference cannot be released.
func (s *Store) Release(id ID) error {
	s.initialize()
	e, exists := s.entries[id]
	if !exists {
		return fmt.Errorf("unknown memory value ID %d", id)
	}
	if id == RootID && e.references == 1 {
		return fmt.Errorf("cannot release the store-owned list root reference")
	}
	released := id
	pending := []ID{id}
	for len(pending) > 0 {
		id := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		e := s.entries[id]
		e.references--
		if e.references != 0 {
			continue
		}
		delete(s.entries, id)
		s.memoryUsage -= entryBytes(e.value)
		switch node := e.value.(type) {
		case ListNode:
			pending = append(pending, node.Prev, node.Value)
		case Some:
			pending = append(pending, node.Element)
		}
	}
	if s.events != nil {
		s.events.Add(robotevent.MemoryReleaseEvent{ID: released})
	}
	return nil
}
