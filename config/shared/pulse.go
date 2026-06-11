package shared

import (
	"fmt"
	"sync"

	"github.com/machanirobotics/pulse/pulse-go"
)

var (
	Pulse *pulse.Pulse // Global Pulse instance for the config package, initialized once.
	once  sync.Once    // Ensures that Pulse is only initialized once, even if New is called multiple times.
)

func init() {
	once.Do(func() {
		p, err := pulse.New().
			WithService("runtime-go-config", "1.0.0").
			WithLogLevel(pulse.ModuleLevel_2).
			Build()
		if err != nil {
			fmt.Printf("ERROR: Failed to create Pulse: %v\n", err)
			panic(err)
		}

		Pulse = p
	})
}

// Close should be called by the main application on shutdown
func Close() error {
	if Pulse != nil {
		return Pulse.Close()
	}
	return nil
}
