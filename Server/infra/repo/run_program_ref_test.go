package repo

import (
	"encoding/json/jsontext"
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

func TestRunProgramReference(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	mustCreateUser(t, pool, "owner", nil)
	if err := NewPrograms(pool).Create(t.Context(), domain.Program{ID: "p", GoogleSubject: "owner", Name: "draft", Source: jsontext.Value(`{}`), TutorialID: ptr("lesson")}); err != nil {
		t.Fatal(err)
	}
	runs := NewRuns(pool)
	cases := []struct {
		tutorial string
		ref      *domain.ProgramRef
	}{
		{tutorial: "lesson"},
		{tutorial: "lesson", ref: &domain.ProgramRef{ID: "p"}},
		{tutorial: "lesson", ref: &domain.ProgramRef{ID: "p", Revision: -1}},
		{tutorial: "lesson", ref: &domain.ProgramRef{ID: "p", Revision: math.MaxInt32 + 1}},
		{tutorial: "lesson", ref: &domain.ProgramRef{Revision: 1}},
		{ref: &domain.ProgramRef{ID: "p", Revision: 1}},
	}
	for _, tc := range cases {
		id, err := runs.Create(t.Context(), domain.Run{GoogleSubject: "owner", FieldID: "field", TutorialID: tc.tutorial, Program: tc.ref})
		if !errors.Is(err, domain.ErrInvalidProgram) || id != "" {
			t.Errorf("Create(%q,%+v)=%q,%v, want invalid reference", tc.tutorial, tc.ref, id, err)
		}
	}
	var count int
	if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM runs").Scan(&count); err != nil {
		t.Errorf("count rejected runs: %v", err)
	} else if count != 0 {
		t.Errorf("rejected references persisted %d runs", count)
	}
	for _, ref := range []*domain.ProgramRef{nil, {ID: "p", Revision: 1}} {
		tutorial := ""
		if ref != nil {
			tutorial = "lesson"
		}
		id, err := runs.Create(t.Context(), domain.Run{GoogleSubject: "owner", FieldID: "field", TutorialID: tutorial, Program: ref})
		if err != nil {
			t.Errorf("Create(%+v): %v", ref, err)
			continue
		}
		got, err := runs.Get(t.Context(), id)
		if err != nil {
			t.Errorf("Get(%s): %v", id, err)
			continue
		}
		if !reflect.DeepEqual(got.Program, ref) {
			t.Errorf("program=%+v, want %+v", got.Program, ref)
		}
	}
}
