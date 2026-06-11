package core

// User represents a system user with basic identification information.
// It provides a cross-platform representation of user data that can be
// populated from platform-specific user management systems.
type User struct {
	// Name is the username or login name of the user.
	Name string
	// ID is the unique identifier for the user, typically the user ID (UID) as a string.
	ID string
}
