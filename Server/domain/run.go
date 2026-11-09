package domain

import "encoding/json/jsontext"

// TutorialRun identifies an execution and the field chosen for it.
type TutorialRun struct {
	ID      string
	FieldID string
}

// Run stores the selected field and the completed result. A nil result is pending.
type Run struct {
	ID              string
	GoogleSubject   string
	OpponentSubject string
	TutorialID      string
	FieldID         string
	ProgramID       string
	Result          jsontext.Value
}
