package domain

import (
	"regexp"
	"time"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

// IsValidUsername reports whether name is 1 to 32 permitted ASCII characters.
func IsValidUsername(name string) bool {
	return usernamePattern.MatchString(name)
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
