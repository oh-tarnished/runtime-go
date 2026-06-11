# ULID Package

A simple, clean ULID (Universally Unique Lexicographically Sortable Identifier) package for Go.

## What is a ULID?

A ULID is a **26-character unique identifier** with two parts:

```
01KB4C0QP014671GWPTAJ1563D
├─────────┬──────────────────
│         └─ Random (16 chars, 80-bit)
└─ Time (10 chars, 48-bit timestamp)
```

- **Time Code** (10 characters): 48-bit timestamp with millisecond precision
- **Random Code** (16 characters): 80-bit randomness for uniqueness

**Benefits:**

- Sortable by creation time
- URL-safe (no special characters)
- Case-insensitive
- More compact than UUID (26 vs 36 characters)

## Quick Start

```go
import "github.com/oh-tarnished/runtime-go/ulid"

// Generate a ULID
id := ulid.Generate()
fmt.Println(id.String())  // "01KB4C0QP014671GWPTAJ1563D"

// Get components
fmt.Println(id.TimeCode())    // "01KB4C0QP0"
fmt.Println(id.RandomCode())   // "14671GWPTAJ1563D"

// Get timestamp
fmt.Println(id.Time())  // 2025-11-28 10:07:21 +0530 IST
```

## Basic Usage

### Generate a ULID

```go
// Simple generation
id := ulid.Generate()

// With error handling
id, err := ulid.New()
if err != nil {
    log.Fatal(err)
}

// As string
idStr := ulid.GenerateString()
```

### Get ULID Components

```go
id := ulid.Generate()

// Full ULID string
full := id.String()  // "01KB4C0QP014671GWPTAJ1563D"

// Time code (48-bit timestamp)
timeCode := id.TimeCode()  // "01KB4C0QP0"

// Random code (80-bit randomness)
randomCode := id.RandomCode()  // "14671GWPTAJ1563D"

// Timestamp as time.Time
timestamp := id.Time()  // time.Time object
```

### Advanced: Timestamp Operations

Use the `Timestamp` accessor for more timestamp operations:

```go
id := ulid.Generate()

// Get timestamp
ts := id.Time()  // time.Time
date := id.Date()  // Date only
unix := id.Time().Unix()  // Unix seconds
```

### Advanced: Duration Operations

Use the `Duration` accessor for duration calculations:

```go
id1 := ulid.Generate()
time.Sleep(100 * time.Millisecond)
id2 := ulid.Generate()

// Age (time since creation)
age := id1.Age()  // time.Duration

// Duration between ULIDs
duration := id1.To(id2)  // time.Duration
```

### Advanced: Stitching ULIDs

Combine the timestamp from one ULID with the random part from another. This is useful when you want to maintain the same random component across different timestamps:

```go
userULID := ulid.Generate()
time.Sleep(100 * time.Millisecond)
frameULID := ulid.Generate()

// Create a new ULID with frame's timestamp and user's random part
stitchedULID := ulid.Stitch(frameULID, userULID)

fmt.Println(stitchedULID.Time())  // Same as frameULID
fmt.Println(stitchedULID.RandomCode())  // Same as userULID
```

**Use Case:** When processing multiple frames for a user, you can generate ULIDs that have unique timestamps (from each frame) but share the user's random component for easy grouping.

## Common Use Cases

### Database IDs

```go
type User struct {
    ID   ulid.ID `json:"id" db:"id"`
    Name string  `json:"name"`
}

user := User{
    ID:   ulid.Generate(),
    Name: "John Doe",
}
```

### Parsing and Validation

```go
// Parse from string
id, err := ulid.Parse("01KB4C0QP014671GWPTAJ1563D")
if err != nil {
    log.Fatal("Invalid ULID")
}

// Validate and get ID in one step
id, err := ulid.Validate("01KB4C0QP014671GWPTAJ1563D")
if err != nil {
    log.Fatal("Invalid ULID")
}
// id is ready to use
fmt.Println(id.String())
```

### Protobuf Integration

**Creating ULID from server timestamp:**

```go
import (
    "google.golang.org/protobuf/types/known/timestamppb"
    "google.golang.org/protobuf/types/known/durationpb"
)

// Receive timestamp from server
serverTimestamp := timestamppb.Now()

// Create ULID from protobuf timestamp
id, err := ulid.FromTimestampPB(serverTimestamp)
if err != nil {
    log.Fatal(err)
}

// Create ULID from protobuf duration (time in the past)
serverDuration := durationpb.New(5 * time.Minute)
id, err := ulid.FromDurationPB(serverDuration)  // Creates ULID for 5 minutes ago
```

**Converting ULID to protobuf:**

```go
id := ulid.Generate()

// Get as timestamppb.Timestamp
timestamp := id.TimestampProto()

// Get age as durationpb.Duration
age := id.AgeProto()

// Send to server
// server.SendTimestamp(timestamp)
// server.SendDuration(age)
```

### Creating from time.Time

```go
// Create ULID from specific time
specificTime := time.Date(2025, 11, 28, 10, 0, 0, 0, time.UTC)
id, err := ulid.FromTime(specificTime)
if err != nil {
    log.Fatal(err)
}
```

### Sorting by Time

```go
id1 := ulid.Generate()
time.Sleep(10 * time.Millisecond)
id2 := ulid.Generate()

// Compare timestamps
if id1.Time().Before(id2.Time()) {
    fmt.Println("id1 was created first")
}
```

### More Usage Patterns

#### Generate strings or pointers

```go
// Generate a plain string when you do not need the ID type.
idString := ulid.GenerateString()

// Generate a pointer when a nullable string field needs to be filled.
idStringRef := ulid.GenerateStringRef()
```

#### Marshal and unmarshal

```go
id := ulid.Generate()

text, err := id.MarshalText()
if err != nil {
    log.Fatal(err)
}

var roundTrip ulid.ID
if err := roundTrip.UnmarshalText(text); err != nil {
    log.Fatal(err)
}

binaryValue, err := id.MarshalBinary()
if err != nil {
    log.Fatal(err)
}

_ = binaryValue
```

#### Build IDs from time values

```go
ts := time.Date(2025, 11, 28, 10, 30, 0, 0, time.UTC)

idFromTime, err := ulid.FromTime(ts)
if err != nil {
    log.Fatal(err)
}

idFromDuration, err := ulid.FromDurationPB(durationpb.New(5 * time.Minute))
if err != nil {
    log.Fatal(err)
}

fmt.Println(idFromTime.Time())
fmt.Println(idFromDuration.Age())
```

#### Build stitched sequences

```go
userID := ulid.Generate()
stitcher := ulid.NewStitcher(userID)

first := stitcher.Next()
second := stitcher.Next()

fmt.Println(first.TimeCode(), first.RandomCode())
fmt.Println(second.TimeCode(), second.RandomCode())
```

## API Reference

### Core Methods

| Method            | Returns     | Description                |
| ----------------- | ----------- | -------------------------- |
| `String()`        | `string`    | Full 26-character ULID     |
| `Time()`          | `time.Time` | Timestamp as time.Time     |
| `TimeCode()`      | `string`    | First 10 chars (timestamp) |
| `RandomCode()`    | `string`    | Last 16 chars (random)     |

### Generation Functions

| Function              | Returns       | Description                               |
| --------------------- | ------------- | ----------------------------------------- |
| `Generate()`          | `ID`          | Generate ULID (panics on error)           |
| `GenerateString()`    | `string`      | Generate as string                        |
| `New()`               | `(ID, error)` | Generate with error handling              |
| `Parse(s)`            | `(ID, error)` | Parse from string                         |
| `Validate(s)`         | `(ID, error)` | Validate and return ID                    |
| `FromTimestampPB(ts)` | `(ID, error)` | Create from protobuf timestamp            |
| `FromDurationPB(dur)` | `(ID, error)` | Create from protobuf duration (past time) |
| `FromTime(t)`         | `(ID, error)` | Create from time.Time                     |

### Advanced Accessors

**Timestamp Operations** (`id.*`):

- `Time()` - Full timestamp
- `Date()` - Date only
- `TimestampProto()` - Protobuf timestamp

**Duration Operations** (`id.*`):

- `Age()` - Time since creation
- `AgeProto()` - Age as `durationpb.Duration`
- `To(other)` - Duration to another ULID
- `ToProto(other)` - Duration to another as protobuf
- `Since(t)` - Duration to specific time
- `SinceProto(t)` - Duration to time as protobuf

**Protobuf Conversions:**

- `id.TimestampProto()` - Get timestamp as `timestamppb.Timestamp`
- `id.AgeProto()` - Get age as `durationpb.Duration`

### Concurrency Safety

ULIDs are safe to generate concurrently from multiple goroutines:

```go
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        id := ulid.Generate()  // Thread-safe
        // All IDs are unique, even when generated simultaneously
    }()
}
wg.Wait()
```

**Why ULIDs avoid race conditions:**

1. **Millisecond precision** (48-bit timestamp)
2. **80-bit random component** for same-millisecond uniqueness
3. **Monotonic entropy** ensures increasing values within same millisecond
4. **Thread-safe** generation using `crypto/rand`

## Examples

See the `examples/` directory:

```bash
# Basic usage
go run examples/basic/main.go

# Validation and parsing
go run examples/validation/main.go

# Protobuf integration (timestamppb, durationpb)
go run examples/protobuf/main.go

# Timestamp operations
go run examples/timestamp/main.go

# Duration calculations
go run examples/duration/main.go

# Component breakdown
go run examples/components/main.go

# Concurrency safety demo
go run examples/concurrency/main.go

# Stitching ULIDs (combining timestamp and random parts)
go run examples/stitch/main.go
```
