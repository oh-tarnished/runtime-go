// Package system provides utilities for interacting with the operating system.
//
// This package contains platform-specific implementations for common system
// operations such as user management and power control. The package uses
// build tags to provide appropriate implementations for different operating
// systems while maintaining a consistent API.
//
// User Management:
//
// The package provides functionality for querying logged-in users and finding
// users with specific capabilities such as audio access:
//
//	// List all logged-in users
//	users, err := system.ListLoggedInUsers()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	for _, user := range users {
//		fmt.Printf("User: %s (ID: %s)\n", user.Name, user.ID)
//	}
//
//	// Find a user capable of audio operations
//	audioUser, found, err := system.GetAudioCapableUser()
//	if err != nil {
//		log.Fatal(err)
//	}
//	if found {
//		fmt.Printf("Audio user: %s\n", audioUser.Name)
//	}
//
// Power Management:
//
// System power operations are provided for controlled shutdown and restart
// of the system. These operations typically require elevated privileges:
//
//	// Reboot the system immediately
//	err := system.Reboot()
//	if err != nil {
//		log.Printf("Failed to reboot: %v", err)
//	}
//
//	// Shutdown the system immediately
//	err = system.Shutdown()
//	if err != nil {
//		log.Printf("Failed to shutdown: %v", err)
//	}
//
// Platform Support:
//
// The package currently provides implementations for Linux systems using
// systemd and standard Unix utilities.
//
// Privileges and Security:
//
// Many operations in this package require elevated privileges to function
// properly. Power management operations typically need root access, while
// user queries may require appropriate permissions to access system user
// databases and session information.
//
// Dependencies:
//
// Linux implementations depend on systemd utilities (loginctl, systemctl)
// and standard Unix tools (shutdown). The package will return appropriate
// errors if required system utilities are not available.
package system
