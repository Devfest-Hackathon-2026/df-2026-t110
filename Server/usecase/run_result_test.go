package usecase

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

type resultRepositoryStub struct {
	run   domain.Run
	err   error
	calls int
}

func (s *resultRepositoryStub) Get(context.Context, string) (domain.Run, error) {
	s.calls++
	return s.run, s.err
}

func TestRunResults(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, subject    string
		result           jsontext.Value
		repoErr, wantErr error
	}{
		{name: "completed", subject: "owner", result: jsontext.Value(`{"history":[]}`)},
		{name: "pending", subject: "owner"},
		{name: "other owner", subject: "other", wantErr: ErrForbidden},
		{name: "unauthenticated", wantErr: ErrUnauthenticated},
		{name: "not found", subject: "owner", repoErr: ErrNotFound, wantErr: ErrNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repository := &resultRepositoryStub{run: domain.Run{GoogleSubject: "owner", Result: tt.result}, err: tt.repoErr}
			result, err := NewRunResults(repository).Result(t.Context(), tt.subject, "run-id")
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Result error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && string(result) != string(tt.result) {
				t.Errorf("result = %s, want %s", result, tt.result)
			}
			if tt.wantErr != nil && len(result) != 0 {
				t.Error("rejected lookup exposed a result")
			}
			if tt.subject == "" && repository.calls != 0 {
				t.Error("unauthenticated lookup accessed storage")
			}
		})
	}
}
