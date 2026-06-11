package main

import (
	"fmt"
	"time"

	"github.com/oh-tarnished/runtime-go/ulid"
)

func main() {
	fmt.Println("=== ULID Components Demo ===")
	fmt.Println()

	// Generate a ULID
	id := ulid.Generate()

	fmt.Println("Full ULID Structure:")
	fmt.Printf("  Complete: %s\n", id.String())
	fmt.Println()

	// Show the two main components
	fmt.Println("ULID Components:")
	fmt.Printf("  Time Code (48-bit):   %s  <- Timestamp with millisecond precision\n", id.TimeCode())
	fmt.Printf("  Random Code (80-bit): %s  <- Randomness for uniqueness\n", id.RandomCode())
	fmt.Println()

	// Show what the time code represents
	fmt.Println("Time Code Details:")
	timestamp := id.Time()
	fmt.Printf("  Timestamp: %s\n", timestamp.Format(time.RFC3339))
	fmt.Printf("  Unix time: %d\n", timestamp.Unix())
	fmt.Printf("  Date only: %s\n", timestamp.Format("2006-01-02"))
	fmt.Println()

	// Generate multiple ULIDs to show time code consistency
	fmt.Println("Multiple ULIDs generated in quick succession:")
	for i := 0; i < 3; i++ {
		id := ulid.Generate()
		fmt.Printf("  %d. %s | Time: %s | Random: %s\n",
			i+1,
			id.String(),
			id.TimeCode(),
			id.RandomCode())
	}
	fmt.Println()

	// Show ULIDs generated with delay
	fmt.Println("ULIDs generated with 100ms delay:")
	for i := 0; i < 3; i++ {
		id := ulid.Generate()
		fmt.Printf("  %d. %s | Time: %s | Random: %s\n",
			i+1,
			id.String(),
			id.TimeCode(),
			id.RandomCode())
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println()

	// Using with Duration and Timestamp accessors
	fmt.Println("Using with Nested Accessors:")
	id1 := ulid.Generate()
	fmt.Printf("  ID: %s\n", id1.String())
	fmt.Printf("  Direct Time(): %s\n", id1.Time().Format(time.RFC3339))
	fmt.Printf("  TimestampProto(): %s\n", id1.TimestampProto().AsTime().Format(time.RFC3339))
	fmt.Printf("  Age(): %v\n", id1.Age())
}
