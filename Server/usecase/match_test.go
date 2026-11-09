package usecase

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

type matchUserStub struct{ name *string }

func (s matchUserStub) Get(context.Context, string) (domain.User, error) {
	return domain.User{GoogleSubject: "owner", Name: s.name}, nil
}

type matchProgramsStub struct{ programs map[string]domain.Program }

func (s matchProgramsStub) Get(_ context.Context, id string) (domain.Program, error) {
	p, ok := s.programs[id]
	if !ok {
		return domain.Program{}, ErrNotFound
	}
	return p, nil
}

type matchOpponentStub struct {
	team       domain.BattleTeam
	err        error
	challenger string
}

func (s *matchOpponentStub) Draw(_ context.Context, challenger string) (domain.BattleTeam, error) {
	s.challenger = challenger
	return s.team, s.err
}

type matchLayoutsStub struct{ layouts []domain.BattleLayout }

func (s matchLayoutsStub) List(context.Context) ([]domain.BattleLayout, error) {
	return s.layouts, nil
}

type matchExecutorStub struct {
	player, opponent []domain.Program
	layout           domain.BattleLayout
	err              error
	calls            int
}

func (s *matchExecutorStub) Execute(_ context.Context, player, opponent []domain.Program, layout domain.BattleLayout) (jsontext.Value, error) {
	s.calls++
	s.player, s.opponent, s.layout = player, opponent, layout
	return jsontext.Value(`{"history":[],"summary":{"score":{"player":1,"opponent":0}}}`), s.err
}

type matchStoreStub struct {
	record domain.MatchRecord
	err    error
	calls  int
}

func (s *matchStoreStub) Create(_ context.Context, record domain.MatchRecord) (domain.Match, error) {
	s.calls++
	s.record = record
	return domain.Match{ID: "match-id", RunID: "run-id", FieldID: record.FieldID, SelectedLayoutID: record.SelectedLayoutID, OpponentName: record.OpponentName}, s.err
}

func TestCreateMatch(t *testing.T) {
	t.Parallel()
	storageErr := errors.New("storage unavailable")
	for _, tt := range []struct {
		name                                  string
		subject                               string
		ids                                   []string
		noName                                bool
		drawErr, executeErr, saveErr, wantErr error
	}{
		{name: "ordered duplicate programs", subject: "owner", ids: []string{"b", "a", "b"}},
		{name: "no session", ids: []string{"a", "a", "a"}, wantErr: ErrUnauthenticated},
		{name: "no display name", subject: "owner", noName: true, ids: []string{"a", "a", "a"}, wantErr: ErrForbidden},
		{name: "empty slots", subject: "owner", wantErr: domain.ErrInvalidProgram},
		{name: "too few slots", subject: "owner", ids: []string{"a"}, wantErr: domain.ErrInvalidProgram},
		{name: "too many slots", subject: "owner", ids: []string{"a", "a", "a", "a"}, wantErr: domain.ErrInvalidProgram},
		{name: "blank ID", subject: "owner", ids: []string{"a", " ", "a"}, wantErr: domain.ErrInvalidProgram},
		{name: "missing program", subject: "owner", ids: []string{"a", "missing", "a"}, wantErr: ErrNotFound},
		{name: "other user's program", subject: "owner", ids: []string{"a", "foreign", "a"}, wantErr: ErrForbidden},
		{name: "tutorial program", subject: "owner", ids: []string{"a", "tutorial", "a"}, wantErr: domain.ErrInvalidProgram},
		{name: "no opponent", subject: "owner", ids: []string{"a", "a", "a"}, drawErr: ErrNoOpponent, wantErr: ErrNoOpponent},
		{name: "draw failure", subject: "owner", ids: []string{"a", "a", "a"}, drawErr: storageErr, wantErr: storageErr},
		{name: "execution failure", subject: "owner", ids: []string{"a", "a", "a"}, executeErr: domain.ErrInvalidProgram, wantErr: domain.ErrInvalidProgram},
		{name: "save failure", subject: "owner", ids: []string{"a", "a", "a"}, saveErr: storageErr, wantErr: storageErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			name, opponentName := "Owner", "Opponent"
			var displayName *string
			if !tt.noName {
				displayName = &name
			}
			opponent := &matchOpponentStub{team: domain.BattleTeam{
				User:     domain.User{GoogleSubject: "opponent", Name: &opponentName},
				Programs: []domain.Program{{ID: "o2"}, {ID: "o1"}, {ID: "o2"}},
			}, err: tt.drawErr}
			executor := &matchExecutorStub{err: tt.executeErr}
			store := &matchStoreStub{err: tt.saveErr}
			service := NewMatches(MatchDeps{
				Users: matchUserStub{displayName}, Programs: matchProgramsStub{map[string]domain.Program{
					"a":        {ID: "a", GoogleSubject: "owner", IsBattle: true},
					"b":        {ID: "b", GoogleSubject: "owner", IsBattle: true},
					"foreign":  {ID: "foreign", GoogleSubject: "other", IsBattle: true},
					"tutorial": {ID: "tutorial", GoogleSubject: "owner"},
				}}, Opponents: opponent,
				Layouts:  matchLayoutsStub{[]domain.BattleLayout{{ID: "layout-a", FieldID: "field"}, {ID: "layout-b", FieldID: "field"}}},
				Executor: executor, Matches: store,
			})
			service.pick = func(n int) int {
				if n != 2 {
					t.Errorf("layout count=%d, want 2", n)
				}
				return 1
			}
			match, err := service.Create(t.Context(), tt.subject, tt.ids)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Create error=%v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if match.ID != "" || (tt.saveErr == nil && store.calls != 0) {
					t.Errorf("failed match was saved or returned: match=%+v, saves=%d", match, store.calls)
				}
				if tt.executeErr == nil && tt.saveErr == nil && executor.calls != 0 {
					t.Error("rejected request executed the engine")
				}
				return
			}
			var gotIDs []string
			for _, p := range executor.player {
				gotIDs = append(gotIDs, p.ID)
			}
			if !reflect.DeepEqual(gotIDs, tt.ids) || !reflect.DeepEqual(executor.opponent, opponent.team.Programs) {
				t.Errorf("engine programs=%v/%v", gotIDs, executor.opponent)
			}
			if opponent.challenger != "owner" || executor.layout.ID != "layout-b" || executor.calls != 1 {
				t.Errorf("draw=%+v, execution=%+v", opponent, executor)
			}
			if match.ID != "match-id" || match.RunID != "run-id" || match.OpponentName != opponentName || match.SelectedLayoutID != "layout-b" {
				t.Errorf("match=%+v", match)
			}
			if store.record.GoogleSubject != "owner" || store.record.OpponentSubject != "opponent" || !store.record.Result.IsValid() {
				t.Errorf("saved record=%+v", store.record)
			}
		})
	}
}
