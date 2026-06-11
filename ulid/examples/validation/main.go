package main

import (
	"fmt"

	"github.com/oh-tarnished/runtime-go/ulid"
)

func main() {
	fmt.Println("=== ULID Validation Demo ===")

	// Generate a ULID first
	original := ulid.Generate()
	ulidString := original.String()
	fmt.Printf("Original ULID: %s\n\n", ulidString)

	// Method 1: Parse (just parsing)
	fmt.Println("Method 1: Parse")
	parsed, err := ulid.Parse(ulidString)
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
	} else {
		fmt.Printf("✓ Parsed: %s\n", parsed.String())
		fmt.Printf("  Time: %s\n", parsed.Time())
	}

	// Method 2: Validate (validates AND returns the ID)
	fmt.Println("\nMethod 2: Validate (recommended)")
	validated, err := ulid.Validate(ulidString)
	if err != nil {
		fmt.Printf("Validation error: %v\n", err)
	} else {
		fmt.Printf("✓ Valid ULID: %s\n", validated.String())
		fmt.Printf("  TimeCode: %s\n", validated.TimeCode())
		fmt.Printf("  RandomCode: %s\n", validated.RandomCode())
	}

	// Test with invalid ULID
	fmt.Println("\n=== Testing Invalid ULID ===")
	invalidULID := "INVALID_ULID_STRING"
	_, err = ulid.Validate(invalidULID)
	if err != nil {
		fmt.Printf("✓ Correctly rejected invalid ULID: %v\n", err)
	}

	// Practical usage: validate user input
	fmt.Println("\n=== Practical Usage ===")
	userInput := ulidString // Simulating user input

	id, err := ulid.Validate(userInput)
	if err != nil {
		fmt.Printf("Invalid ULID from user: %v\n", err)
		return
	}

	// Now you can use the ID directly
	fmt.Printf("Processing user's ULID: %s\n", id.String())
	fmt.Printf("Created at: %s\n", id.Time())
	fmt.Printf("Age: %v\n", id.Age())
}
