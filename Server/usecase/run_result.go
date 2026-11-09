package usecase

import (
	"context"
	"encoding/json/jsontext"
	"fmt"
	"strings"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

type RunResultRepository interface {
	Get(ctx context.Context, id string) (domain.Run, error)
}

type RunResults struct{ runs RunResultRepository }

func NewRunResults(runs RunResultRepository) *RunResults {
	return &RunResults{runs: runs}
}

// Result returns the saved result, or nil while pending, to a run participant.
func (r *RunResults) Result(ctx context.Context, subject, id string) (jsontext.Value, error) {
	if strings.TrimSpace(subject) == "" {
		return nil, ErrUnauthenticated
	}
	run, err := r.runs.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get run: %w", err)
	}
	if run.GoogleSubject != subject && run.OpponentSubject != subject {
		return nil, ErrForbidden
	}
	return run.Result, nil
}
