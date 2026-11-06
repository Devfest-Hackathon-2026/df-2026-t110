package memory

import (
	"math"
	"testing"
)

func TestReferenceCountOverflowIsAtomic(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		rootCount uint64
		operation func(*Store, ID) error
	}{
		{"retain primitive", 1, func(s *Store, id ID) error { return s.Retain(id) }},
		{"retain root", math.MaxUint64, func(s *Store, _ ID) error { return s.Retain(RootID) }},
		{"add root", math.MaxUint64, func(s *Store, _ ID) error {
			_, err := s.Add(ListRoot{})
			return err
		}},
		{"previous edge", math.MaxUint64, func(s *Store, id ID) error {
			_, err := s.Add(ListNode{Prev: RootID, Value: id})
			return err
		}},
		{"value edge", 1, func(s *Store, id ID) error {
			_, err := s.Add(ListNode{Prev: RootID, Value: id})
			return err
		}},
		{"two equal edges", math.MaxUint64 - 1, func(s *Store, _ ID) error {
			_, err := s.Add(ListNode{Prev: RootID, Value: RootID})
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var s Store
			id, err := s.Add(Unit{})
			if err != nil {
				t.Fatalf("prepare primitive: %v", err)
			}
			s.entries[id].references = math.MaxUint64
			s.entries[RootID].references = tc.rootCount
			if err := tc.operation(&s, id); err == nil {
				t.Error("operation accepted reference count overflow")
			}
			if len(s.entries) != 2 || s.nextID != id {
				t.Errorf("failed operation changed allocations: %+v", s)
			}
			if s.entries[id].references != math.MaxUint64 || s.entries[RootID].references != tc.rootCount {
				t.Error("failed operation changed reference counts")
			}
		})
	}
}

func TestIDExhaustionDoesNotRetainEdges(t *testing.T) {
	t.Parallel()
	var s Store
	s.initialize()
	s.nextID = math.MaxUint64
	if _, err := s.Add(ListNode{Prev: RootID, Value: RootID}); err == nil {
		t.Error("Add accepted exhausted IDs")
	}
	if len(s.entries) != 1 || s.entries[RootID].references != 1 {
		t.Error("failed allocation retained outgoing edges")
	}
}
