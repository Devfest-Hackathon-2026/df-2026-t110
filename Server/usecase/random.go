package usecase

import (
	"crypto/rand"
	"encoding/base64"
)

// newRandomID returns 256 bits of randomness, which is unguessable enough to
// serve directly as a bearer secret such as a session ID.
func newRandomID() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
