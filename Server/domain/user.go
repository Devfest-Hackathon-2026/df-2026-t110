package domain

import "time"

// User is an account of this service, identified at Google by GoogleSubject.
type User struct {
	ID            string
	GoogleSubject string
	Name          string
}

// Session ties a browser session cookie to a user.
type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
}

// Expired reports whether the session is no longer valid at now; it expires
// exactly at ExpiresAt.
func (s Session) Expired(now time.Time) bool {
	return !now.Before(s.ExpiresAt)
}
