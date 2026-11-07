package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

type fakeSubmissions struct {
	subs []domain.Submission
	err  error
}

func (f fakeSubmissions) LatestByField(context.Context, string) ([]domain.Submission, error) {
	return f.subs, f.err
}

type fakeLayouts struct {
	ids []string
	err error
}

func (f fakeLayouts) LayoutIDs(context.Context, string) ([]string, error) { return f.ids, f.err }

func TestMatchingDraw(t *testing.T) {
	t.Parallel()
	me := domain.Submission{GoogleSubject: "me", ProgramID: "p-me"}
	other := domain.Submission{GoogleSubject: "other", ProgramID: "p-other"}
	last := func(n int) int { return n - 1 }

	tests := []struct {
		name    string
		subs    fakeSubmissions
		layouts fakeLayouts
		want    domain.Pairing
		err     error
	}{
		{"draws layout and opponent", fakeSubmissions{subs: []domain.Submission{me, other}}, fakeLayouts{ids: []string{"l1", "l2"}}, domain.Pairing{LayoutID: "l2", Opponent: other}, nil},
		{"no opponent", fakeSubmissions{subs: []domain.Submission{me}}, fakeLayouts{ids: []string{"l1"}}, domain.Pairing{}, ErrNoOpponent},
		{"field without layouts", fakeSubmissions{subs: []domain.Submission{other}}, fakeLayouts{}, domain.Pairing{}, ErrNotFound},
		{"unknown field", fakeSubmissions{}, fakeLayouts{err: ErrNotFound}, domain.Pairing{}, ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewMatching(tt.subs, tt.layouts)
			m.pick = last
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
