package main

import (
	"fmt"

	"github.com/oh-tarnished/runtime-go/ulid"
)

func main() {
	// Generate a user ULID
	userULID := ulid.Generate()
	fmt.Printf("User ULID:        %s\n", userULID.String())
	fmt.Printf("  Timestamp:      %v\n", userULID.Time())
	fmt.Printf("  Random Part:    %s\n\n", userULID.RandomCode())

	// Simulate generating multiple frame ULIDs with different timestamps
	// but keeping the same random part from the user ULID
	fmt.Println("Generating 3 frame ULIDs with user's random part:")

	// Create a Stitcher that automatically handles timestamp increments
	stitcher := ulid.NewStitcher(userULID)

	for i := 0; i < 3; i++ {
		// Simply call Next() - it automatically increments the timestamp
		stitchedULID := stitcher.Next()

		fmt.Printf("\nFrame %d:\n", i+1)
		fmt.Printf("  Stitched ULID:   %s\n", stitchedULID.String())
		fmt.Printf("    Timestamp:     %v\n", stitchedULID.Time())
		fmt.Printf("    Random Part:   %s (same as user)\n", stitchedULID.RandomCode())
	}

	fmt.Println("\n✓ All stitched ULIDs have different timestamps but share the user's random part")
}
