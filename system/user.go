package system

import (
	"fmt"
	"runtime"

	"github.com/oh-tarnished/runtime-go/system/core"
	"github.com/oh-tarnished/runtime-go/system/linux"
)

// GetAudioCapableUser retrieves the first logged-in user with audio capabilities.
func GetAudioCapableUser() (core.User, bool, error) {
	switch runtime.GOOS {
	case "linux":
		return linux.GetAudioCapableUser()
	default:
		return core.User{}, false, fmt.Errorf("audio capable user retrieval is not implemented for %s", runtime.GOOS)
	}
}

// ListLoggedInUsers returns a list of all currently logged-in users on the system.
func ListLoggedInUsers() ([]core.User, error) {
	switch runtime.GOOS {
	case "linux":
		return linux.ListLoggedInUsers()
	default:
		return nil, fmt.Errorf("logged in users retrieval is not implemented for %s", runtime.GOOS)
	}
}
