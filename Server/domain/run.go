package domain

import "encoding/json/jsontext"

// TutorialRun identifies an execution and the field chosen for it.
type TutorialRun struct {
	ID      string
	FieldID string
}

// Job is what an executor simulates for a run: a TutorialJob or a MatchJob.
type Job interface{ isJob() }

// TutorialJob runs one program on the chosen variant of a tutorial.
type TutorialJob struct {
	Program  Program
	Tutorial Tutorial
	Variant  int
}

// MatchJob runs the challenger's team against the opponent's on layout, rated from Entry.
type MatchJob struct {
	Entry    MatchEntry
	Player   []Program
	Opponent []Program
	Layout   BattleLayout
}

func (TutorialJob) isJob() {}
func (MatchJob) isJob()    {}

// Run stores the selected field and the completed result. A nil result is pending.
type Run struct {
	Program         *ProgramRef // nil for match runs
	ID              string
	GoogleSubject   string
	OpponentSubject string
	TutorialID      string
	FieldID         string
	Result          jsontext.Value
}
