package usecase

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"slices"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

type programRepositoryStub struct {
	saved      []domain.Program
	updated    []domain.Program
	program    domain.Program
	getID      string
	getErr     error
	listed     []domain.Program
	tutorialID string
	subject    string
	err        error
}

func (s *programRepositoryStub) Get(_ context.Context, id string) (domain.Program, error) {
	s.getID = id
	return s.program, s.getErr
}

func (s *programRepositoryStub) Update(_ context.Context, p domain.Program) (domain.Program, error) {
	s.updated = append(s.updated, p)
	return p, s.err
}

func (s *programRepositoryStub) Create(_ context.Context, p domain.Program) error {
	s.saved = append(s.saved, p)
	return s.err
}

func (s *programRepositoryStub) GetByTutorialID(_ context.Context, subject, tutorialID string) ([]domain.Program, error) {
	s.tutorialID = tutorialID
	s.subject = subject
	return s.listed, s.err
}

type sourceValidatorStub struct{ err error }

func (s sourceValidatorStub) ValidateSource(context.Context, []byte) error { return s.err }

func TestProgramsCreate(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, subject                   string
		validationErr, repoErr, wantErr error
	}{
		{name: "create", subject: "owner"},
		{name: "no subject", wantErr: ErrUnauthenticated},
		{name: "syntax error", subject: "owner", validationErr: domain.ErrInvalidProgram, wantErr: domain.ErrInvalidProgram},
		{name: "validator failure", subject: "owner", validationErr: context.Canceled, wantErr: context.Canceled},
		{name: "repository failure", subject: "owner", repoErr: context.DeadlineExceeded, wantErr: context.DeadlineExceeded},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := &programRepositoryStub{err: tt.repoErr}
			uc := NewPrograms(store, sourceValidatorStub{err: tt.validationErr})
			draft := domain.Program{ID: "untrusted-id", GoogleSubject: "attacker", Name: "draft", IsBattle: true, Source: jsontext.Value(`{}`)}
			got, err := uc.Create(context.Background(), tt.subject, draft)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if tt.repoErr == nil && len(store.saved) != 0 {
					t.Error("saved invalid draft")
				}
				return
			}
			if got.ID == "" || got.ID == draft.ID || got.GoogleSubject != "owner" {
				t.Errorf("incorrect identity: %+v", got)
			}
			second, err := uc.Create(context.Background(), tt.subject, draft)
			if err != nil {
				t.Errorf("second create: %v", err)
			}
			if second.ID == got.ID {
				t.Error("IDs are not unique")
			}
		})
	}
}

func TestProgramsListByTutorialID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		programs []domain.Program
		err      error
	}{
		{name: "matches", programs: []domain.Program{{ID: "draft-1"}, {ID: "draft-2"}}},
		{name: "no matches"},
		{name: "repository failure", err: context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store := &programRepositoryStub{listed: tc.programs, err: tc.err}
			programs := NewPrograms(store, sourceValidatorStub{})
			got, err := programs.ListByTutorialID(t.Context(), "owner", "tutorial-1")
			if !errors.Is(err, tc.err) {
				t.Errorf("error = %v, want %v", err, tc.err)
			}
			if store.subject != "owner" || store.tutorialID != "tutorial-1" {
				t.Errorf("repository arguments = (%q, %q), want (owner, tutorial-1)", store.subject, store.tutorialID)
			}
			if !slices.EqualFunc(got, tc.programs, func(got, want domain.Program) bool { return got.ID == want.ID }) {
				t.Errorf("programs = %+v, want %+v", got, tc.programs)
			}
		})
	}
}

func TestProgramsUpdate(t *testing.T) {
	t.Parallel()
	store := &programRepositoryStub{program: domain.Program{ID: "p", GoogleSubject: "owner", IsBattle: true}}
	service := NewPrograms(store, sourceValidatorStub{})
	got, err := service.Update(t.Context(), "owner", "p", domain.Program{Name: "edited", Source: jsontext.Value(`{}`)})
	if err != nil {
		t.Errorf("update: %v", err)
	}
	if !got.IsBattle || got.ID != "p" || got.GoogleSubject != "owner" {
		t.Errorf("changed identity: %+v", got)
	}
}
