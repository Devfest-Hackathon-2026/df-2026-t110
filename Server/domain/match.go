package domain

import "errors"

// ErrNoOpponent reports that nobody except the challenger has submitted a program to the field.
var ErrNoOpponent = errors.New("no opponent candidate")

// Submission is the program a user most recently updated on a field.
type Submission struct {
	GoogleSubject string
	ProgramID     string
}

// Pairing is the draw result of a ranked match: who the challenger faces and on which layout.
type Pairing struct {
	LayoutID string
	Opponent Submission
}

// SelectOpponent picks one submission not owned by challenger.
//
// pick returns an index in [0, n) and is injected so the draw is deterministic in tests.
// The challenger is excluded before drawing so that every eligible opponent is equally likely.
func SelectOpponent(challenger string, submissions []Submission, pick func(n int) int) (Submission, error) {
	candidates := make([]Submission, 0, len(submissions))
	for _, s := range submissions {
		if s.GoogleSubject != challenger {
			candidates = append(candidates, s)
		}
	}
	if len(candidates) == 0 {
		return Submission{}, ErrNoOpponent
	}
	return candidates[pick(len(candidates))], nil
}
