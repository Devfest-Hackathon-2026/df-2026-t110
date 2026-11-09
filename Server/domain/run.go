package domain

import "encoding/json/jsontext"

// TutorialRun identifies an execution and the field chosen for it.
type TutorialRun struct {
	ID      string
	FieldID string
}

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
