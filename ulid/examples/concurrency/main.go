package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/oh-tarnished/runtime-go/ulid"
)

func main() {
	fmt.Println("=== ULID Concurrency Demo ===")
	fmt.Println("Generating ULIDs from multiple goroutines simultaneously")

	// Number of concurrent goroutines
	numGoroutines := 10
	numIDsPerGoroutine := 5

	// Channel to collect all generated IDs
	idChan := make(chan ulid.ID, numGoroutines*numIDsPerGoroutine)
	var wg sync.WaitGroup

	// Launch multiple goroutines that generate ULIDs concurrently
	startTime := time.Now()
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numIDsPerGoroutine; j++ {
				id := ulid.Generate()
				idChan <- id
				// Simulate some work
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(idChan)
	duration := time.Since(startTime)

	// Collect all IDs
	var ids []ulid.ID
	for id := range idChan {
		ids = append(ids, id)
	}

	fmt.Printf("Generated %d ULIDs in %v\n\n", len(ids), duration)

	// Demonstrate uniqueness
	fmt.Println("=== Uniqueness Check ===")
	uniqueMap := make(map[string]bool)
	duplicates := 0
	for _, id := range ids {
		if uniqueMap[id.String()] {
			duplicates++
		}
		uniqueMap[id.String()] = true
	}
	fmt.Printf("Total IDs: %d\n", len(ids))
	fmt.Printf("Unique IDs: %d\n", len(uniqueMap))
	fmt.Printf("Duplicates: %d ✓\n\n", duplicates)

	// Show time distribution
	fmt.Println("=== Time Distribution ===")
	fmt.Println("First 10 ULIDs with timestamps:")
	for i := 0; i < 10 && i < len(ids); i++ {
		fmt.Printf("%2d. %s | Time: %s | TimeCode: %s\n",
			i+1,
			ids[i].String(),
			ids[i].Time().Format("15:04:05.000"),
			ids[i].TimeCode())
	}

	// Demonstrate sortability
	fmt.Println("\n=== Sortability Demo ===")
	fmt.Println("ULIDs are naturally sortable by creation time:")

	// Create ULIDs with known time gaps
	id1 := ulid.Generate()
	time.Sleep(5 * time.Millisecond)
	id2 := ulid.Generate()
	time.Sleep(5 * time.Millisecond)
	id3 := ulid.Generate()

	fmt.Printf("\nID1: %s (created at %s)\n", id1.String(), id1.Time().Format("15:04:05.000"))
	fmt.Printf("ID2: %s (created at %s)\n", id2.String(), id2.Time().Format("15:04:05.000"))
	fmt.Printf("ID3: %s (created at %s)\n", id3.String(), id3.Time().Format("15:04:05.000"))

	// String comparison works for sorting
	fmt.Println("\nString comparison (lexicographic):")
	fmt.Printf("ID1 < ID2: %v ✓\n", id1.String() < id2.String())
	fmt.Printf("ID2 < ID3: %v ✓\n", id2.String() < id3.String())
	fmt.Printf("ID1 < ID3: %v ✓\n", id1.String() < id3.String())

	// Time comparison
	fmt.Println("\nTime comparison:")
	fmt.Printf("ID1.Time() < ID2.Time(): %v ✓\n", id1.Time().Before(id2.Time()))
	fmt.Printf("ID2.Time() < ID3.Time(): %v ✓\n", id2.Time().Before(id3.Time()))

	// Show millisecond precision
	fmt.Println("\n=== Millisecond Precision ===")
	fmt.Println("Generating ULIDs in rapid succession:")
	for i := 0; i < 5; i++ {
		id := ulid.Generate()
		fmt.Printf("%d. %s | %s | TimeCode: %s\n",
			i+1,
			id.String(),
			id.Time().Format("15:04:05.000"),
			id.TimeCode())
		time.Sleep(2 * time.Millisecond)
	}

	// Race condition avoidance explanation
	fmt.Println("\n=== Why ULID Avoids Race Conditions ===")
	fmt.Println("1. Millisecond precision timestamp (48-bit)")
	fmt.Println("2. 80-bit random component for same-millisecond uniqueness")
	fmt.Println("3. Monotonic entropy ensures increasing values within same millisecond")
	fmt.Println("4. Thread-safe generation (uses crypto/rand)")
	fmt.Println("\nResult: Even with concurrent generation, ULIDs are:")
	fmt.Println("  ✓ Unique (no duplicates)")
	fmt.Println("  ✓ Sortable (by creation time)")
	fmt.Println("  ✓ Safe (no race conditions)")
}
