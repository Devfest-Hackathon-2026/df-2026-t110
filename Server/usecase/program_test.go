package usecase

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

type programRepositoryStub struct {
	saved []domain.Program
	err   error
}

func (s *programRepositoryStub) Create(_ context.Context, p domain.Program) error {
	s.saved = append(s.saved, p)
	return s.err
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
