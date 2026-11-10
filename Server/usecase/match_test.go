package usecase

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/google/uuid"
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
	candidates []domain.OpponentCandidate
	team       domain.BattleTeam
	err        error
	challenger string
}

func (s *matchOpponentStub) SelectOpponent(_ context.Context, challenger string, choose func([]domain.OpponentCandidate) (domain.OpponentCandidate, error)) (domain.BattleTeam, error) {
	s.challenger = challenger
	if s.err != nil {
		return domain.BattleTeam{}, s.err
	}
	chosen, err := choose(s.candidates)
	if err != nil {
		return domain.BattleTeam{}, err
	}
	team := s.team
	team.User.GoogleSubject, team.Rating = chosen.GoogleSubject, chosen.Rating
	return team, nil
}

type matchLayoutsStub struct{ layouts []domain.BattleLayout }

func (s matchLayoutsStub) List(context.Context) ([]domain.BattleLayout, error) {
	return s.layouts, nil
}

type matchRatingStub struct{ rating int }

func (s matchRatingStub) Rating(context.Context, string) (int, error) { return s.rating, nil }

func TestCreateMatch(t *testing.T) {
	t.Parallel()
	storageErr := errors.New("storage unavailable")
	for _, tt := range []struct {
		name                                            string
		subject                                         string
		ids                                             []string
		noName                                          bool
		selectionErr, simulateErr, completeErr, wantErr error
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
		{name: "no opponent", subject: "owner", ids: []string{"a", "a", "a"}, selectionErr: ErrNoOpponent, wantErr: ErrNoOpponent},
		{name: "selection failure", subject: "owner", ids: []string{"a", "a", "a"}, selectionErr: storageErr, wantErr: storageErr},
		{name: "simulation failure", subject: "owner", ids: []string{"a", "a", "a"}, simulateErr: domain.ErrInvalidProgram, wantErr: domain.ErrInvalidProgram},
		{name: "result save failure", subject: "owner", ids: []string{"a", "a", "a"}, completeErr: storageErr, wantErr: storageErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			name, opponentName := "Owner", "Opponent"
			var displayName *string
			if !tt.noName {
				displayName = &name
			}
			opponent := &matchOpponentStub{candidates: []domain.OpponentCandidate{{GoogleSubject: "opponent", Rating: 1200}}, team: domain.BattleTeam{
				User:     domain.User{GoogleSubject: "opponent", Name: &opponentName},
				Programs: []domain.Program{{ID: "o2", Revision: 9}, {ID: "o1", Revision: 4}, {ID: "o2", Revision: 9}},
				Rating:   1200,
			}, err: tt.selectionErr}
			runs := &runRepositoryStub{}
			results := &matchResultRepositoryStub{err: tt.completeErr}
			simulator := &matchSimulatorStub{err: tt.simulateErr}
			service := NewMatches(MatchDeps{
				Users: matchUserStub{displayName}, Programs: matchProgramsStub{map[string]domain.Program{
					"a":        {ID: "a", Revision: 2, GoogleSubject: "owner", IsBattle: true},
					"b":        {ID: "b", Revision: 5, GoogleSubject: "owner", IsBattle: true},
					"foreign":  {ID: "foreign", GoogleSubject: "other", IsBattle: true},
					"tutorial": {ID: "tutorial", GoogleSubject: "owner"},
				}}, Opponents: opponent,
				Layouts:  matchLayoutsStub{[]domain.BattleLayout{{FieldID: "layout-a"}, {FieldID: "layout-b"}}},
				Executor: NewExecutor(runs, results, nil, simulator),
				Matches:  matchRatingStub{1000},
			})
			service.pick = func(n int) int {
				if n != 2 {
					t.Errorf("layout count=%d, want 2", n)
				}
				return 1
			}
			matchID := uuid.Must(uuid.NewV7())
			service.newID = func() (uuid.UUID, error) { return matchID, nil }
			match, err := service.Create(t.Context(), tt.subject, tt.ids)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Create error=%v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if match.ID != "" {
					t.Errorf("failed match was returned: %+v", match)
				}
				if tt.simulateErr == nil && tt.completeErr == nil && runs.created != nil {
					t.Error("rejected request created a run")
				}
				return
			}
			job := simulator.job
			var gotIDs []string
			for _, p := range job.Player {
				gotIDs = append(gotIDs, p.ID)
			}
			if !reflect.DeepEqual(gotIDs, tt.ids) || !reflect.DeepEqual(job.Opponent, opponent.team.Programs) {
				t.Errorf("engine programs=%v/%v", gotIDs, job.Opponent)
			}
			if opponent.challenger != "owner" || job.Layout.FieldID != "layout-b" || simulator.calls != 1 {
				t.Errorf("selection=%+v, simulated %d times on %q", opponent, simulator.calls, job.Layout.FieldID)
			}
			if run := runs.created; run == nil || run.GoogleSubject != "owner" || run.FieldID != "layout-b" || run.TutorialID != "" || run.Program != nil {
				t.Errorf("executed run=%+v", run)
			}
			if match.ID != matchID.String() || match.RunID != "run-id" || match.OpponentName != opponentName || match.FieldID != "layout-b" {
				t.Errorf("match=%+v", match)
			}
			want := domain.MatchEntry{ID: matchID.String(), GoogleSubject: "owner", OpponentSubject: "opponent", OpponentName: opponentName, ChallengerRating: 1000, OpponentRating: 1200}
			wantRecord := want.Record(domain.MatchResult{})
			wantRecord.ChallengerPrograms = []domain.ProgramRef{{ID: "b", Revision: 5}, {ID: "a", Revision: 2}, {ID: "b", Revision: 5}}
			wantRecord.OpponentPrograms = []domain.ProgramRef{{ID: "o2", Revision: 9}, {ID: "o1", Revision: 4}, {ID: "o2", Revision: 9}}
			if results.completed == nil || !reflect.DeepEqual(*results.completed, wantRecord) {
				t.Errorf("completed=%+v, want %+v", results.completed, wantRecord)
			}
		})
	}
}

func TestCreateMatchChoosesOpponentByRating(t *testing.T) {
	t.Parallel()

	// Relative to a challenger rated 1000, near weighs exp(-0.22) and far exp(-8), so far
	// owns only the lowest 0.04% of draws; a uniform pick would give far draws below 0.5.
	// A challenger rated 1600 reverses the shares.
	candidates := []domain.OpponentCandidate{
		{GoogleSubject: "owner", Rating: 1000},
		{GoogleSubject: "far", Rating: 1600},
		{GoogleSubject: "near", Rating: 1100},
	}

	cases := []struct {
		name       string
		rating     int
		candidates []domain.OpponentCandidate
		draw       float64
		want       string
		wantErr    error
	}{
		{name: "closer rating takes most draws", rating: 1000, candidates: candidates, draw: 0.4, want: "near"},
		{name: "weights follow the challenger rating", rating: 1600, candidates: candidates, draw: 0.5, want: "far"},
		{name: "challenger is never chosen", rating: 1000, candidates: candidates[:1], draw: 0, wantErr: ErrNoOpponent},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			name, opponentName := "Owner", "Opponent"
			opponent := &matchOpponentStub{candidates: tt.candidates, team: domain.BattleTeam{
				User:     domain.User{Name: &opponentName},
				Programs: []domain.Program{{ID: "o"}, {ID: "o"}, {ID: "o"}},
			}}
			runs, results := &runRepositoryStub{}, &matchResultRepositoryStub{}
			service := NewMatches(MatchDeps{
				Users:     matchUserStub{&name},
				Programs:  matchProgramsStub{map[string]domain.Program{"a": {ID: "a", GoogleSubject: "owner", IsBattle: true}}},
				Opponents: opponent,
				Layouts:   matchLayoutsStub{[]domain.BattleLayout{{FieldID: "field"}}},
				Executor:  NewExecutor(runs, results, nil, &matchSimulatorStub{}),
				Matches:   matchRatingStub{tt.rating},
			})
			service.draw = func() float64 { return tt.draw }

			_, err := service.Create(t.Context(), "owner", []string{"a", "a", "a"})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Create error=%v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if runs.created != nil {
					t.Error("failed matching created a run")
				}
				return
			}
			if results.completed == nil || results.completed.OpponentSubject != tt.want {
				t.Errorf("completed=%+v, want opponent %q", results.completed, tt.want)
			}
		})
	}
}
