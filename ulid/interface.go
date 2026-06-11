package ulid

// Generator defines the minimal generation and parsing surface used by the
// package-level convenience helpers.
type Generator interface {
	// New generates a ULID using the current time and secure entropy.
	New() (ID, error)
	// NewString generates a ULID string using the current time and secure entropy.
	NewString() (string, error)
	// Parse validates and parses a ULID string.
	Parse(s string) (ID, error)
	// Validate checks a ULID string and returns the parsed value.
	Validate(s string) (ID, error)
}
