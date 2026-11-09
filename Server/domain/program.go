package domain

import (
	"encoding/json/jsontext"
	"errors"
	"strings"
)

var ErrInvalidProgram = errors.New("invalid program")

// Program is a saved draft owned by a verified user. Source retains exact JSON
// numbers, including integers that cannot be represented as float64.
type Program struct {
	Revision      int32
	ID            string
	GoogleSubject string
	Name          string
	Source        jsontext.Value
	IsBattle      bool
	TutorialID    *string
}

func (p Program) Validate() error {
	if strings.TrimSpace(p.Name) == "" || !p.Source.IsValid() {
		return ErrInvalidProgram
	}
	if p.IsBattle {
		if p.TutorialID != nil {
			return ErrInvalidProgram
		}
	} else if p.TutorialID == nil || strings.TrimSpace(*p.TutorialID) == "" {
		return ErrInvalidProgram
	}
	return nil
}
