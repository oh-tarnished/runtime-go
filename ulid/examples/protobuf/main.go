package main

import (
	"fmt"
	"time"

	"github.com/oh-tarnished/runtime-go/ulid"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	fmt.Println("=== ULID with Protobuf Demo ===")

	// Scenario 1: Receive timestamppb.Timestamp from server
	fmt.Println("\n--- Creating ULID from Protobuf Timestamp ---")

	// Simulate receiving a timestamp from your server
	serverTimestamp := timestamppb.New(time.Now().Add(-1 * time.Hour))
	fmt.Printf("Server timestamp: %s\n", serverTimestamp.AsTime().Format(time.RFC3339))

	// Create ULID from the protobuf timestamp
	id, err := ulid.FromTimestampPB(serverTimestamp)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created ULID: %s\n", id.String())
	fmt.Printf("ULID time: %s\n", id.Time().Format(time.RFC3339))

	// Scenario 2: Create ULID from time.Time
	fmt.Println("\n--- Creating ULID from time.Time ---")

	specificTime := time.Date(2025, 11, 28, 10, 0, 0, 0, time.UTC)
	id3, err := ulid.FromTime(specificTime)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created ULID: %s\n", id3.String())
	fmt.Printf("ULID time: %s\n", id3.Time().Format(time.RFC3339))

	// Scenario 3: Convert ULID to protobuf types
	fmt.Println("\n--- Converting ULID to Protobuf ---")

	currentID := ulid.Generate()
	fmt.Printf("Current ULID: %s\n", currentID.String())

	// Get timestamp as protobuf
	pbTimestamp := currentID.TimestampProto()
	fmt.Printf("As timestamppb: %s\n", pbTimestamp.AsTime().Format(time.RFC3339))

	// Get age as protobuf duration
	pbDuration := currentID.AgeProto()
	fmt.Printf("Age as durationpb: %v\n", pbDuration.AsDuration())

	// Scenario 4: Working with durationpb from server
	fmt.Println("\n--- Creating ULID from Protobuf Duration ---")

	// Simulate receiving a duration from server (e.g., "create ID for 5 minutes ago")
	serverDuration := durationpb.New(5 * time.Minute)
	fmt.Printf("Server duration: %v\n", serverDuration.AsDuration())

	// Create ULID from duration (calculates time in the past)
	idFromDuration, err := ulid.FromDurationPB(serverDuration)
	if err != nil {
		panic(err)
	}
	fmt.Printf("ULID from %v ago: %s\n", serverDuration.AsDuration(), idFromDuration.String())
	fmt.Printf("ULID time: %s\n", idFromDuration.Time().Format(time.RFC3339))
	fmt.Printf("Age: %v\n", idFromDuration.Age())

	// Scenario 5: Complete workflow
	fmt.Println("\n--- Complete Server Workflow ---")

	// 1. Receive timestamp from server
	incomingTimestamp := timestamppb.Now()
	fmt.Printf("1. Received from server: %s\n", incomingTimestamp.AsTime().Format(time.RFC3339))

	// 2. Create ULID from server timestamp
	workflowID, err := ulid.FromTimestampPB(incomingTimestamp)
	if err != nil {
		panic(err)
	}
	fmt.Printf("2. Created ULID: %s\n", workflowID.String())

	// 3. Do some work...
	time.Sleep(10 * time.Millisecond)

	// 4. Calculate processing duration and send back as protobuf
	processingDuration := workflowID.AgeProto()
	fmt.Printf("3. Processing took: %v\n", processingDuration.AsDuration())

	// 5. Send timestamp back as protobuf
	responseTimestamp := workflowID.TimestampProto()
	fmt.Printf("4. Sending back timestamp: %s\n", responseTimestamp.AsTime().Format(time.RFC3339))

	fmt.Println("\n✓ All protobuf conversions working!")
}
