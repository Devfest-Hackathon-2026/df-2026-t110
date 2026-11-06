// Package usecase implements application use cases and declares the ports
// (repositories and external services) they depend on.
package usecase

import "errors"

var (
	// ErrNotFound reports that the requested entity does not exist.
	ErrNotFound = errors.New("not found")
	// ErrUnauthenticated reports that the request has no valid session.
	ErrUnauthenticated = errors.New("unauthenticated")
	// ErrInvalidLogin reports a rejected, expired, replayed, or unverifiable login attempt.
	ErrInvalidLogin = errors.New("invalid login")
	// ErrLoginUnavailable reports that no identity provider is configured.
	ErrLoginUnavailable = errors.New("login unavailable")
	// ErrInvalidUsername reports a name outside the allowed format.
	ErrInvalidUsername = errors.New("invalid username")
	// ErrUsernameTaken reports a name assigned to another user.
	ErrUsernameTaken = errors.New("username already taken")
	// ErrUsernameAlreadySet reports that the user's name cannot be changed.
	ErrUsernameAlreadySet = errors.New("username already set")
)
