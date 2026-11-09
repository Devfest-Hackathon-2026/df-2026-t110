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

func TestTutorialRuns(t *testing.T) {
	t.Parallel()
	lessonID, otherLesson := "tutorial-1", "tutorial-2"
	valid := domain.Program{ID: "program-1", GoogleSubject: "owner", TutorialID: &lessonID}
	storageErr := errors.New("database unavailable")
	for _, tt := range []struct {
		name                             string
		subject                          string
		program                          domain.Program
		tutorialErr, programErr, wantErr error
		simulateErr, saveErr             error
	}{
		{name: "selected field", subject: "owner", program: valid},
		{name: "unauthenticated", program: valid, wantErr: ErrUnauthenticated},
		{name: "other owner", subject: "other", program: valid, wantErr: ErrForbidden},
		{name: "battle program", subject: "owner", program: domain.Program{GoogleSubject: "owner", IsBattle: true}, wantErr: domain.ErrInvalidProgram},
		{name: "other tutorial", subject: "owner", program: domain.Program{GoogleSubject: "owner", TutorialID: &otherLesson}, wantErr: domain.ErrInvalidProgram},
		{name: "missing tutorial", subject: "owner", tutorialErr: ErrNotFound, wantErr: ErrNotFound},
		{name: "missing program", subject: "owner", programErr: ErrNotFound, wantErr: ErrNotFound},
		{name: "storage failure", subject: "owner", programErr: storageErr, wantErr: storageErr},
		{name: "simulation failure", subject: "owner", program: valid, simulateErr: domain.ErrInvalidProgram, wantErr: domain.ErrInvalidProgram},
		{name: "result save failure", subject: "owner", program: valid, saveErr: storageErr, wantErr: storageErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runs := &runRepositoryStub{saveErr: tt.saveErr}
			simulator := &tutorialSimulatorStub{result: jsontext.Value(`{"history":[]}`), err: tt.simulateErr}
			lesson := runTutorialsStub{
				tutorial: domain.Tutorial{ID: lessonID, Stage: schema.StageDefinition{Variants: []schema.FieldDefinition{{Id: "field-a"}, {Id: "field-b"}}}},
				err:      tt.tutorialErr,
			}
			service := NewTutorialRuns(runProgramsStub{tt.program, tt.programErr}, lesson, NewExecutor(runs, nil, simulator, nil))
			service.pick = func(n int) int {
				if n != 2 {
					t.Errorf("variant count = %d, want 2", n)
				}
				return 1
			}
			got, err := service.Run(t.Context(), tt.subject, lessonID, "program-1")
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Run error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if runs.created != nil && tt.simulateErr == nil && tt.saveErr == nil {
					t.Error("rejected request created a run")
				}
				if got.ID != "" {
					t.Errorf("failed run returned an ID: %s", got.ID)
				}
				return
			}
			job := simulator.job
			if got.ID != "run-id" || got.FieldID != "field-b" || simulator.calls != 1 || job.Variant != 1 || job.Program.ID != "program-1" || job.Tutorial.ID != lessonID {
				t.Errorf("run = %+v, simulated %d times with %+v", got, simulator.calls, job)
			}
			if run := runs.created; run == nil || run.GoogleSubject != "owner" || run.TutorialID != lessonID || run.FieldID != got.FieldID || run.ProgramID != "program-1" || run.Result != nil {
				t.Errorf("created run = %+v", run)
			}

		})
	}
}
