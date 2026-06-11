package linux

import (
	"fmt"
	"os/exec"
)

// Reboot initiates an immediate system reboot using the shutdown command.
// This function requires appropriate privileges (typically root) to execute successfully.
// The reboot is immediate and will not wait for graceful application shutdown.
func Reboot() error {
	if err := exec.Command("shutdown", "-r", "now").Run(); err != nil {
		return fmt.Errorf("failed to reboot: %w", err)
	}

	return nil
}

// Shutdown initiates an immediate system shutdown using the shutdown command.
// This function requires appropriate privileges (typically root) to execute successfully.
// The shutdown is immediate and will not wait for graceful application shutdown.
func Shutdown() error {
	if err := exec.Command("shutdown", "now").Run(); err != nil {
		return fmt.Errorf("failed to shutdown: %w", err)
	}

	return nil
}
