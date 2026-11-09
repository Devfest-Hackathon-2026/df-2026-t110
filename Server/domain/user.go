package domain

import (
	"regexp"
	"strings"
	"time"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

// IsValidUsername reports whether name has a permitted format and is not reserved.
func IsValidUsername(name string) bool {
	return usernamePattern.MatchString(name) && !strings.EqualFold(name, "self") && !strings.EqualFold(name, "std")
}

// User is an account of this service, identified at Google by GoogleSubject.
type User struct {
	GoogleSubject string
	Name          *string
}

// Session ties a browser session cookie to a user.
type Session struct {
	ID            string
	GoogleSubject string
	ExpiresAt     time.Time
}

// Expired reports whether the session is no longer valid at now; it expires
// exactly at ExpiresAt.
func (s Session) Expired(now time.Time) bool {
	return !now.Before(s.ExpiresAt)
}
