package ulid

import (
	"crypto/rand"
	"time"

	okulid "github.com/oklog/ulid/v2"
)

type generator struct{}

// New generates a ULID using the current time and secure entropy.
func (g *generator) New() (ID, error) {
	entropy := okulid.Monotonic(rand.Reader, 0)
	u, err := okulid.New(okulid.Timestamp(time.Now()), entropy)
	if err != nil {
		return ID{}, err
	}
	return NewID(u), nil
}

// NewString generates a ULID and returns it as a string.
func (g *generator) NewString() (string, error) {
	id, err := g.New()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// Parse validates and parses a ULID string.
func (g *generator) Parse(s string) (ID, error) {
	u, err := okulid.Parse(s)
	if err != nil {
		return ID{}, err
	}
	return NewID(u), nil
}

// Validate checks a ULID string and returns the parsed ID.
func (g *generator) Validate(s string) (ID, error) {
	return g.Parse(s)
}
