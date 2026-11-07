package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

type fakeSubmissions struct {
	subs        []domain.Submission
	err         error
	gotFieldIDs *[]string
}

func (f fakeSubmissions) LatestByField(_ context.Context, fieldID string) ([]domain.Submission, error) {
	if f.gotFieldIDs != nil {
		*f.gotFieldIDs = append(*f.gotFieldIDs, fieldID)
	}
	return f.subs, f.err
}

type fakeLayouts struct {
	ids         []string
	err         error
	gotFieldIDs *[]string
}

func (f fakeLayouts) LayoutIDs(_ context.Context, fieldID string) ([]string, error) {
	if f.gotFieldIDs != nil {
		*f.gotFieldIDs = append(*f.gotFieldIDs, fieldID)
	}
	return f.ids, f.err
}

func TestMatchingDraw(t *testing.T) {
	t.Parallel()
	me := domain.Submission{GoogleSubject: "me", ProgramID: "p-me"}
	o1 := domain.Submission{GoogleSubject: "o1", ProgramID: "p-o1"}
	o2 := domain.Submission{GoogleSubject: "o2", ProgramID: "p-o2"}
	errRepo := errors.New("repository failure")
	first := func(int) int { return 0 }
	last := func(n int) int { return n - 1 }

	tests := []struct {
		name    string
		subs    fakeSubmissions
		layouts fakeLayouts
		pick    func(int) int
		want    domain.Pairing
		err     error
	}{
		{"first picks: first layout and first opponent", fakeSubmissions{subs: []domain.Submission{me, o1, o2}}, fakeLayouts{ids: []string{"l1", "l2"}}, first, domain.Pairing{LayoutID: "l1", Opponent: o1}, nil},
		{"last picks: last layout and last opponent", fakeSubmissions{subs: []domain.Submission{me, o1, o2}}, fakeLayouts{ids: []string{"l1", "l2"}}, last, domain.Pairing{LayoutID: "l2", Opponent: o2}, nil},
		{"no opponent", fakeSubmissions{subs: []domain.Submission{me}}, fakeLayouts{ids: []string{"l1"}}, last, domain.Pairing{}, ErrNoOpponent},
		{"field without layouts", fakeSubmissions{subs: []domain.Submission{o1}}, fakeLayouts{}, last, domain.Pairing{}, ErrNotFound},
		{"unknown field", fakeSubmissions{}, fakeLayouts{err: ErrNotFound}, last, domain.Pairing{}, ErrNotFound},
		{"submission lookup fails", fakeSubmissions{err: errRepo}, fakeLayouts{ids: []string{"l1"}}, last, domain.Pairing{}, errRepo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewMatching(tt.subs, tt.layouts)
			m.pick = tt.pick
			got, err := m.Draw(context.Background(), "f", "me")
			if !errors.Is(err, tt.err) {
				t.Errorf("err = %v, want %v", err, tt.err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestMatchingDrawQueriesRepositoriesWithFieldID(t *testing.T) {
	t.Parallel()
	var subsFieldIDs, layoutFieldIDs []string
	subs := fakeSubmissions{subs: []domain.Submission{{GoogleSubject: "other"}}, gotFieldIDs: &subsFieldIDs}
	layouts := fakeLayouts{ids: []string{"l1"}, gotFieldIDs: &layoutFieldIDs}

	if _, err := NewMatching(subs, layouts).Draw(context.Background(), "field-1", "me"); err != nil {
		t.Fatal(err)
	}

	if len(subsFieldIDs) != 1 || subsFieldIDs[0] != "field-1" {
		t.Errorf("submissions queried with %v, want [field-1]", subsFieldIDs)
	}
	if len(layoutFieldIDs) != 1 || layoutFieldIDs[0] != "field-1" {
		t.Errorf("layouts queried with %v, want [field-1]", layoutFieldIDs)
	}
}
