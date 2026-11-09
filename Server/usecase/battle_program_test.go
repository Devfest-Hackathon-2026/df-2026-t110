package usecase

import (
	"context"
	"errors"
	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"reflect"
	"testing"
)

type battleStore struct{ ids []string }

func (s *battleStore) Set(_ context.Context, _ string, ids []string) error {
	s.ids = append([]string(nil), ids...)
	return nil
}

type battleLookup struct {
	program domain.Program
	err     error
}

func (s battleLookup) Get(_ context.Context, _ string) (domain.Program, error) {
	return s.program, s.err
}
func TestSetBattlePrograms(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name            string
		ids             []string
		owner           string
		battle          bool
		lookupErr, want error
	}{
		{"duplicates", []string{"a", "a", "a"}, "owner", true, nil, nil},
		{"order", []string{"c", "a", "b"}, "owner", true, nil, nil},
		{"two", []string{"a", "b"}, "owner", true, nil, domain.ErrInvalidProgram},
		{"four", []string{"a", "b", "c", "d"}, "owner", true, nil, domain.ErrInvalidProgram},
		{"empty", nil, "owner", true, nil, domain.ErrInvalidProgram},
		{"blank", []string{" ", "a", "b"}, "owner", true, nil, domain.ErrInvalidProgram},
		{"other owner", []string{"a", "b", "c"}, "other", true, nil, ErrForbidden},
		{"tutorial", []string{"a", "b", "c"}, "owner", false, nil, domain.ErrInvalidProgram},
		{"missing", []string{"a", "b", "c"}, "owner", true, ErrNotFound, ErrNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := &battleStore{}
			svc := NewBattlePrograms(store, battleLookup{domain.Program{GoogleSubject: tt.owner, IsBattle: tt.battle}, tt.lookupErr})
			err := svc.Set(context.Background(), "owner", tt.ids)
			if !errors.Is(err, tt.want) {
				t.Errorf("error=%v, want %v", err, tt.want)
			}
			if tt.want == nil && !reflect.DeepEqual(store.ids, tt.ids) {
				t.Errorf("saved=%v, want %v", store.ids, tt.ids)
			}
			if tt.want != nil && store.ids != nil {
				t.Error("invalid request saved")
			}
			if err := svc.Set(context.Background(), "", tt.ids); !errors.Is(err, ErrUnauthenticated) {
				t.Errorf("unauthenticated error=%v", err)
			}
		})
	}
}

func (s *battleStore) Get(_ context.Context, _ string) ([]string, error) { return s.ids, nil }

func TestGetBattlePrograms(t *testing.T) {
	t.Parallel()
	store := &battleStore{ids: []string{"c", "a", "c"}}
	svc := NewBattlePrograms(store, battleLookup{})
	ids, err := svc.Get(context.Background(), "owner")
	if err != nil || !reflect.DeepEqual(ids, store.ids) {
		t.Errorf("Get=%v, %v", ids, err)
	}
	if _, err := svc.Get(context.Background(), ""); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("error=%v", err)
	}
}
