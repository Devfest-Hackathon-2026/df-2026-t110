package repo

import (
	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"math"
	"strings"
)

// Keep PostgreSQL's integer representation at the repository boundary.
func programRevisionToDB(ref domain.ProgramRef) (int32, error) {
	if strings.TrimSpace(ref.ID) == "" || ref.Revision <= 0 || ref.Revision > math.MaxInt32 {
		return 0, domain.ErrInvalidProgram
	}
	return int32(ref.Revision), nil
}
