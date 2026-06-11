package main

import (
	"fmt"

	"github.com/oh-tarnished/runtime-go/ulid"
)

func main() {
	// Generate a ULID
	id := ulid.Generate()
	fmt.Printf("Generated ULID: %s\n\n", id.String())

	// Get ULID components
	fmt.Printf("Time Code:   %s (48-bit timestamp)\n", id.TimeCode())
	fmt.Printf("Random Code: %s (80-bit randomness)\n\n", id.RandomCode())

	// Get timestamp
	fmt.Printf("Timestamp: %s\n", id.Time())
}
