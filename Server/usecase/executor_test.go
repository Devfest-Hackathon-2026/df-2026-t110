package usecase

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

type runRepositoryStub struct {
	created            *domain.Run
	savedID            string
	saved              jsontext.Value
	createErr, saveErr error
}

func (s *runRepositoryStub) Create(_ context.Context, run domain.Run) (string, error) {
	s.created = &run
	if s.createErr != nil {
		return "", s.createErr
	}
	return "run-id", nil
}

func (s *runRepositoryStub) SaveResult(_ context.Context, id string, result jsontext.Value) error {
	s.savedID, s.saved = id, result
	return s.saveErr
}

type matchResultRepositoryStub struct {
	runID     string
	completed *domain.MatchRecord
	err       error
}

func (s *matchResultRepositoryStub) Complete(_ context.Context, runID string, record domain.MatchRecord) error {
	s.runID, s.completed = runID, &record
	return s.err
}

type tutorialSimulatorStub struct {
	calls  int
	job    domain.TutorialJob
	result jsontext.Value
	err    error
}

func (s *tutorialSimulatorStub) Simulate(_ context.Context, program domain.Program, tutorial domain.Tutorial, variant int) (jsontext.Value, error) {
	s.calls++
	s.job = domain.TutorialJob{Program: program, Tutorial: tutorial, Variant: variant}
	return s.result, s.err
}

type matchSimulatorStub struct {
	calls  int
	job    domain.MatchJob
	result domain.MatchResult
	err    error
}

func (s *matchSimulatorStub) Simulate(_ context.Context, player, opponent []domain.Program, layout domain.BattleLayout) (domain.MatchResult, error) {
	s.calls++
	s.job = domain.MatchJob{Player: player, Opponent: opponent, Layout: layout}
	return s.result, s.err
}

func TestExecutorTutorial(t *testing.T) {
	t.Parallel()

	storageErr := errors.New("database unavailable")
	run := domain.Run{GoogleSubject: "owner", TutorialID: "tutorial-1", FieldID: "field-a", Program: &domain.ProgramRef{ID: "program-1", Revision: 1}}
	job := domain.TutorialJob{Program: domain.Program{ID: "program-1"}, Tutorial: domain.Tutorial{ID: "tutorial-1"}, Variant: 1}
	result := jsontext.Value(`{"history":[]}`)

	cases := []struct {
		name                            string
		createErr, simulateErr, saveErr error
		wantErr                         error
		wantSimulated, wantSaved        bool
	}{
		{name: "saves result under created run", wantSimulated: true, wantSaved: true},
		{name: "create failure skips simulation", createErr: storageErr, wantErr: storageErr},
		{name: "simulation failure leaves run pending", simulateErr: domain.ErrInvalidProgram, wantErr: domain.ErrInvalidProgram, wantSimulated: true},
		{name: "save failure", saveErr: storageErr, wantErr: storageErr, wantSimulated: true, wantSaved: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runs := &runRepositoryStub{createErr: tt.createErr, saveErr: tt.saveErr}
			matches := &matchResultRepositoryStub{}
			simulator := &tutorialSimulatorStub{result: result, err: tt.simulateErr}
			executor := NewExecutor(runs, matches, simulator, &matchSimulatorStub{})

			id, err := executor.Execute(t.Context(), run, job)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Execute error = %v, want %v", err, tt.wantErr)
			}

			if runs.created == nil || !reflect.DeepEqual(*runs.created, run) {
				t.Errorf("created run = %+v, want %+v", runs.created, run)
			}
			if simulated := simulator.calls == 1; simulated != tt.wantSimulated {
				t.Errorf("simulated = %t, want %t", simulated, tt.wantSimulated)
			}
			if tt.wantSimulated && (simulator.job.Program.ID != job.Program.ID || simulator.job.Tutorial.ID != job.Tutorial.ID || simulator.job.Variant != job.Variant) {
				t.Errorf("simulated job = %+v, want %+v", simulator.job, job)
			}
			if saved := runs.saved != nil; saved != tt.wantSaved {
				t.Errorf("result saved = %t, want %t", saved, tt.wantSaved)
			}
			if tt.wantSaved && (runs.savedID != "run-id" || string(runs.saved) != string(result)) {
				t.Errorf("saved %s under %q, want %s under run-id", runs.saved, runs.savedID, result)
			}
			if matches.completed != nil {
				t.Errorf("tutorial completed a match: %+v", matches.completed)
			}
			wantID := ""
			if tt.wantErr == nil {
				wantID = "run-id"
			}
			if id != wantID {
				t.Errorf("Execute ID = %q, want %q", id, wantID)
			}
		})
	}
}

func TestExecutorMatch(t *testing.T) {
	t.Parallel()

	storageErr := errors.New("database unavailable")
	run := domain.Run{GoogleSubject: "owner", FieldID: "layout-a"}
	entry := domain.MatchEntry{
		ID: "match-id", GoogleSubject: "owner", OpponentSubject: "opponent", OpponentName: "Opponent",
		ChallengerRating: 1000, OpponentRating: 1200,
	}
	job := domain.MatchJob{
		Entry:    entry,
		Player:   []domain.Program{{ID: "a", Revision: 2}, {ID: "b", Revision: 5}, {ID: "a", Revision: 2}},
		Opponent: []domain.Program{{ID: "o", Revision: 9}, {ID: "o", Revision: 9}, {ID: "o", Revision: 9}},
		Layout:   domain.BattleLayout{FieldID: "layout-a"},
	}
	result := domain.MatchResult{ChallengerScore: 2, OpponentScore: 1, Replay: jsontext.Value(`{"history":[]}`)}
	wantRecord := entry.Record(result)
	wantRecord.ChallengerPrograms = []domain.ProgramRef{{ID: "a", Revision: 2}, {ID: "b", Revision: 5}, {ID: "a", Revision: 2}}
	wantRecord.OpponentPrograms = []domain.ProgramRef{{ID: "o", Revision: 9}, {ID: "o", Revision: 9}, {ID: "o", Revision: 9}}

	cases := []struct {
		name                     string
		simulateErr, completeErr error
		wantErr                  error
		wantCompleted            bool
	}{
		{name: "completes the rated match under created run", wantCompleted: true},
		{name: "simulation failure leaves run pending", simulateErr: domain.ErrInvalidProgram, wantErr: domain.ErrInvalidProgram},
		{name: "complete failure", completeErr: storageErr, wantErr: storageErr, wantCompleted: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runs := &runRepositoryStub{}
			matches := &matchResultRepositoryStub{err: tt.completeErr}
			executor := NewExecutor(runs, matches, &tutorialSimulatorStub{}, &matchSimulatorStub{result: result, err: tt.simulateErr})

			id, err := executor.Execute(t.Context(), run, job)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Execute error = %v, want %v", err, tt.wantErr)
			}

			if runs.created == nil || !reflect.DeepEqual(*runs.created, run) {
				t.Errorf("created run = %+v, want %+v", runs.created, run)
			}
			if runs.saved != nil {
				t.Errorf("match saved a bare result %s", runs.saved)
			}
			if completed := matches.completed != nil; completed != tt.wantCompleted {
				t.Fatalf("match completed = %t, want %t", completed, tt.wantCompleted)
			}
			if tt.wantCompleted && (matches.runID != "run-id" || !reflect.DeepEqual(*matches.completed, wantRecord)) {
				t.Errorf("completed run %q with %+v, want %+v", matches.runID, matches.completed, wantRecord)
			}
			wantID := ""
			if tt.wantErr == nil {
				wantID = "run-id"
			}
			if id != wantID {
				t.Errorf("Execute ID = %q, want %q", id, wantID)
			}
		})
	}
}
