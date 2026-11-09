package memory

import (
	"fmt"
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
		{"some edge", 1, func(s *Store, id ID) error {
			_, err := s.Add(Some{Element: id})
			return err
		}},
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
			before := s.MemoryUsage()
			if err := tc.operation(&s, id); err == nil {
				t.Error("operation accepted reference count overflow")
			}
			if len(s.entries) != 2 || s.nextID != id {
				t.Errorf("failed operation changed allocations: %+v", s)
			}
			if s.entries[id].references != math.MaxUint64 || s.entries[RootID].references != tc.rootCount {
				t.Error("failed operation changed reference counts")
			}
			if got := s.MemoryUsage(); got != before {
				t.Errorf("failed operation changed memory usage from %d to %d", before, got)
			}
		})
	}
}

func TestIDExhaustionDoesNotRetainEdges(t *testing.T) {
	t.Parallel()
	var s Store
	s.initialize()
	s.nextID = math.MaxUint64
	before := s.MemoryUsage()
	if _, err := s.Add(ListNode{Prev: RootID, Value: RootID}); err == nil {
		t.Error("Add accepted exhausted IDs")
	}
	if len(s.entries) != 1 || s.entries[RootID].references != 1 {
		t.Error("failed allocation retained outgoing edges")
	}
	if got := s.MemoryUsage(); got != before {
		t.Errorf("failed allocation changed memory usage from %d to %d", before, got)
	}
}

func TestMemoryUsageOverflowIsAtomic(t *testing.T) {
	t.Parallel()
	for _, value := range []Value{Unit{}, None{}, Some{Element: RootID}, ListNode{Prev: RootID, Value: RootID}} {
		t.Run(fmt.Sprintf("%T", value), func(t *testing.T) {
			t.Parallel()
			var s Store
			s.initialize()
			s.memoryUsage = math.MaxUint64
			if _, err := s.Add(value); err == nil {
				t.Error("Add accepted memory usage overflow")
			}
			if len(s.entries) != 1 || s.nextID != 0 || s.entries[RootID].references != 1 {
				t.Error("failed allocation changed entries, IDs, or reference counts")
			}
			if got := s.MemoryUsage(); got != math.MaxUint64 {
				t.Errorf("failed allocation changed memory usage to %d", got)
			}
			if _, err := s.Add(ListRoot{}); err != nil {
				t.Errorf("retaining the shared root allocated memory: %v", err)
			}
		})
	}
}
