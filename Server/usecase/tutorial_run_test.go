package usecase

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
)

type runProgramsStub struct {
	program domain.Program
	err     error
}

func (s runProgramsStub) Get(context.Context, string) (domain.Program, error) {
	return s.program, s.err
}

type runTutorialsStub struct {
	TutorialRepository
	tutorial domain.Tutorial
	err      error
}

func (s runTutorialsStub) Get(context.Context, string) (domain.Tutorial, error) {
	return s.tutorial, s.err
}

type executorStub struct {
	calls   int
	variant int
	program domain.Program
	err     error
}

func (s *executorStub) Execute(_ context.Context, program domain.Program, _ domain.Tutorial, variant int) (jsontext.Value, error) {
	s.calls++
	s.program, s.variant = program, variant
	return jsontext.Value(`{"history":[],"summary":{"cleared":true,"tick":1,"reason":"condition_met"}}`), s.err
}

type runStoreStub struct {
	run                domain.Run
	createErr, saveErr error
}

func (s *runStoreStub) Create(_ context.Context, run domain.Run) (string, error) {
	run.ID = "run-id"
	s.run = run
	return run.ID, s.createErr
}
func (s *runStoreStub) SaveResult(_ context.Context, _ string, result jsontext.Value) error {
	s.run.Result = result
	return s.saveErr
}

func TestTutorialRuns(t *testing.T) {
	t.Parallel()
	lessonID, otherLesson := "tutorial-1", "tutorial-2"
	valid := domain.Program{ID: "program-1", Revision: 7, GoogleSubject: "owner", TutorialID: &lessonID}
	storageErr := errors.New("database unavailable")
	for _, tt := range []struct {
		name                             string
		subject                          string
		program                          domain.Program
		tutorialErr, programErr, wantErr error
		createErr, saveErr, executeErr   error
	}{
		{name: "selected field", subject: "owner", program: valid},
		{name: "unauthenticated", program: valid, wantErr: ErrUnauthenticated},
		{name: "other owner", subject: "other", program: valid, wantErr: ErrForbidden},
		{name: "battle program", subject: "owner", program: domain.Program{GoogleSubject: "owner", IsBattle: true}, wantErr: domain.ErrInvalidProgram},
		{name: "other tutorial", subject: "owner", program: domain.Program{GoogleSubject: "owner", TutorialID: &otherLesson}, wantErr: domain.ErrInvalidProgram},
		{name: "missing tutorial", subject: "owner", tutorialErr: ErrNotFound, wantErr: ErrNotFound},
		{name: "missing program", subject: "owner", programErr: ErrNotFound, wantErr: ErrNotFound},
		{name: "storage failure", subject: "owner", programErr: storageErr, wantErr: storageErr},
		{name: "create failure", subject: "owner", program: valid, createErr: storageErr, wantErr: storageErr},
		{name: "execution failure", subject: "owner", program: valid, executeErr: domain.ErrInvalidProgram, wantErr: domain.ErrInvalidProgram},
		{name: "result save failure", subject: "owner", program: valid, saveErr: storageErr, wantErr: storageErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			executor := &executorStub{err: tt.executeErr}
			store := &runStoreStub{createErr: tt.createErr, saveErr: tt.saveErr}
			runs := NewTutorialRuns(runProgramsStub{tt.program, tt.programErr}, runTutorialsStub{
				tutorial: domain.Tutorial{ID: lessonID, Stage: schema.StageDefinition{Variants: []schema.FieldDefinition{{Id: "field-a"}, {Id: "field-b"}}}},
				err:      tt.tutorialErr,
			}, executor, store)
			runs.pick = func(n int) int {
				if n != 2 {
					t.Errorf("variant count = %d, want 2", n)
				}
				return 1
			}
			got, err := runs.Run(t.Context(), tt.subject, lessonID, "program-1")
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Run error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if executor.calls != 0 && tt.executeErr == nil && tt.saveErr == nil {
					t.Error("rejected request started an execution")
				}
				if got.ID != "" {
					t.Errorf("failed run returned an ID: %s", got.ID)
				}
				return
			}
			if got.ID != "run-id" || got.FieldID != "field-b" || executor.calls != 1 || executor.variant != 1 || executor.program.ID != "program-1" {
				t.Errorf("run = %+v, execution = %+v", got, executor)
			}
			if store.run.GoogleSubject != "owner" || store.run.TutorialID != lessonID || store.run.FieldID != got.FieldID || store.run.Program == nil || store.run.Program.ID != "program-1" || store.run.Program.Revision != 7 || len(store.run.Result) == 0 {
				t.Errorf("run returned before storing its result: %+v", store.run)
			}

		})
	}
}
