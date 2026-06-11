package main

import (
	"fmt"
	"time"

	"github.com/oh-tarnished/runtime-go/ulid"
)

func main() {
	// Generate first ULID
	id1 := ulid.Generate()
	fmt.Printf("First ULID:  %s\n", id1.String())

	// Wait
	time.Sleep(100 * time.Millisecond)

	// Generate second ULID
	id2 := ulid.Generate()
	fmt.Printf("Second ULID: %s\n\n", id2.String())

	// Calculate duration between ULIDs
	duration := id1.To(id2)
	fmt.Printf("Duration between: %v\n", duration)

	// Calculate age
	age := id1.Age()
	fmt.Printf("Age of first:     %v\n", age)
}
