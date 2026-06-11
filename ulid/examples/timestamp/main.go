package main

import (
	"fmt"
	"time"

	"github.com/oh-tarnished/runtime-go/ulid"
)

func main() {
	// Generate a ULID
	id := ulid.Generate()
	fmt.Printf("ULID: %s\n\n", id.String())

	// Access timestamp using Timestamp accessor
	timestamp := id.Time()
	fmt.Printf("Timestamp: %s\n", timestamp.Format(time.RFC3339))
	fmt.Printf("Date only: %s\n", id.Date().Format("2006-01-02"))

	// Or use direct Time() method
	fmt.Printf("\nDirect Time(): %s\n", id.Time().Format(time.RFC3339))
}
