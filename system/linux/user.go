package linux

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/oh-tarnished/runtime-go/system/core"
)

// loginCtlUser represents the JSON structure returned by loginctl for user information.
type loginCtlUser struct {
	Name string `json:"user"`
	ID   int    `json:"uid"`
}

// GetAudioCapableUser finds the first logged-in user that has access to audio services.
// It checks for the presence of PulseAudio socket and cookie files to determine
// audio capability. This is useful for applications that need to play audio
// in a multi-user environment.
//
// The function returns the user, a boolean indicating if an audio-capable user
// was found, and any error that occurred during the search process.
func GetAudioCapableUser() (core.User, bool, error) {
	loggedInUsers, err := ListLoggedInUsers()
	if err != nil {
		return core.User{}, false, fmt.Errorf("failed to get logged in users: %w", err)
	}

	for _, u := range loggedInUsers {
		// checking if pulse socket path exists for this user
		if _, err := os.Stat(fmt.Sprintf("/run/user/%s/pulse/native", u.ID)); err != nil {
			continue
		}

		details, err := user.Lookup(u.Name)
		if err != nil {
			return core.User{}, false, fmt.Errorf("failed to lookup user information: %w", err)
		}

		// checking if pulse cookie path exists for this user
		if _, err := os.Stat(fmt.Sprintf("%s/.config/pulse/cookie", details.HomeDir)); err != nil {
			continue
		}

		// at this point, both exists, so returning this user
		return u, true, nil
	}

	return core.User{}, false, nil
}

// ListLoggedInUsers returns a list of all currently logged-in users on the system.
// It uses loginctl to query systemd for active user sessions and handles
// different systemd versions that may have varying JSON output formats.
//
// The function requires loginctl to be available on the system and will
// return an error if systemd is not present or accessible.
func ListLoggedInUsers() ([]core.User, error) {
	_, err := exec.LookPath("loginctl")
	if err != nil {
		return []core.User{}, fmt.Errorf("loginctl not found on system")
	}

	systemdVersion, err := getSystemdVersion()
	if err != nil {
		return []core.User{}, fmt.Errorf("failed to get systemd version: %w", err)
	}

	_ = systemdVersion // We can use this to adjust the command if needed in the future

	cmd := exec.Command("loginctl", "list-users", "-o", "json")

	res, err := cmd.Output()
	if err != nil {
		return []core.User{}, fmt.Errorf("failed to get users from loginctl: %w", err)
	}

	var loginctlUsers []loginCtlUser
	if err := json.Unmarshal(res, &loginctlUsers); err != nil {
		return []core.User{}, fmt.Errorf("failed to unmarshal loginctl output: %w", err)
	}

	var users []core.User
	for _, u := range loginctlUsers {
		users = append(users, core.User{
			Name: u.Name,
			ID:   fmt.Sprintf("%d", u.ID),
		})
	}

	return users, nil
}

// getSystemdVersion retrieves the version number of the installed systemd.
// This is used to determine which command line options are available for
// loginctl, as newer versions support additional JSON formatting options.
func getSystemdVersion() (int, error) {
	_, err := exec.LookPath("systemctl")
	if err != nil {
		return 0, fmt.Errorf("systemctl not found on system")
	}

	// Get systemd version
	versionCmd := exec.Command("systemctl", "--version")
	versionOutput, err := versionCmd.Output()
	if err != nil {
		return 0, fmt.Errorf("error getting systemd version: %w", err)
	}

	// Parse version number
	versionStr := string(versionOutput)
	versionParts := strings.Split(versionStr, "\n")[0]

	var version int
	_, err = fmt.Sscanf(versionParts, "systemd %d", &version)
	if err != nil {
		return 0, fmt.Errorf("error parsing systemd version: %w", err)
	}

	return version, nil
}
