package system

import (
	"fmt"
	"runtime"

	"github.com/oh-tarnished/runtime-go/system/linux"
)

// Reboot initiates a system reboot.
func Reboot() error {
	switch runtime.GOOS {
	case "linux":
		return linux.Reboot()
	default:
		return fmt.Errorf("reboot not supported on %s", runtime.GOOS)
	}
}

// Shutdown initiates a system shutdown.
func Shutdown() error {
	switch runtime.GOOS {
	case "linux":
		return linux.Shutdown()
	default:
		return fmt.Errorf("shutdown not supported on %s", runtime.GOOS)
	}
}
