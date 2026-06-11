package ulid

import (
	"crypto/rand"
	"fmt"
	"time"

	okulid "github.com/oklog/ulid/v2"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var defaultGenerator Generator = &generator{}

// New generates a ULID using the package's default generator.
func New() (ID, error) {
	return defaultGenerator.New()
}

// NewString generates a ULID string using the package's default generator.
func NewString() (string, error) {
	return defaultGenerator.NewString()
}

// Parse validates and parses a ULID string.
func Parse(s string) (ID, error) {
	return defaultGenerator.Parse(s)
}

// Validate checks a ULID string and returns the parsed ID.
func Validate(s string) (ID, error) {
	return defaultGenerator.Validate(s)
}

// Generate creates a ULID and panics if generation fails.
func Generate() ID {
	id, err := New()
	if err != nil {
		panic(fmt.Sprintf("ulid: failed to generate: %v", err))
	}
	return id
}

// GenerateString creates a ULID string and panics if generation fails.
func GenerateString() string {
	return Generate().String()
}

// GenerateStringRef creates a ULID string and returns a pointer to it.
func GenerateStringRef() *string {
	id := GenerateString()
	return &id
}

// FromTimestampPB creates a ULID from a protobuf timestamp.
func FromTimestampPB(ts *timestamppb.Timestamp) (ID, error) {
	if ts == nil {
		return ID{}, fmt.Errorf("ulid: timestamp is nil")
	}

	return FromTime(ts.AsTime())
}

// FromDurationPB creates a ULID from a protobuf duration relative to now.
func FromDurationPB(dur *durationpb.Duration) (ID, error) {
	if dur == nil {
		return ID{}, fmt.Errorf("ulid: duration is nil")
	}

	return FromTime(time.Now().Add(-dur.AsDuration()))
}

// FromTime creates a ULID from a specific time.
func FromTime(t time.Time) (ID, error) {
	entropy := okulid.Monotonic(rand.Reader, 0)
	u, err := okulid.New(okulid.Timestamp(t), entropy)
	if err != nil {
		return ID{}, err
	}
	return NewID(u), nil
}

// Stitch combines the timestamp of one ULID with the random code of another.
func Stitch(timestampSource, randomSource ID) ID {
	stitched := timestampSource.TimeCode() + randomSource.RandomCode()
	id, err := Parse(stitched)
	if err != nil {
		panic(fmt.Sprintf("ulid: failed to stitch: %v", err))
	}
	return id
}

// StitchWithOffset creates a stitched ULID at a fixed offset from a base time.
func StitchWithOffset(baseTime time.Time, offset time.Duration, randomSource ID) ID {
	frameTime := baseTime.Add(offset)
	frameULID, err := FromTime(frameTime)
	if err != nil {
		panic(fmt.Sprintf("ulid: failed to create frame ULID: %v", err))
	}
	return Stitch(frameULID, randomSource)
}

// Stitcher generates stitched ULIDs with incrementing timestamps.
type Stitcher struct {
	randomSource ID
	baseTime     time.Time
	counter      int
	increment    time.Duration
}

// NewStitcher creates a Stitcher that increments timestamps by one nanosecond.
func NewStitcher(randomSource ID) *Stitcher {
	return &Stitcher{
		randomSource: randomSource,
		baseTime:     time.Now(),
		counter:      0,
		increment:    time.Nanosecond,
	}
}

// NewStitcherWithIncrement creates a Stitcher with a custom increment.
func NewStitcherWithIncrement(randomSource ID, increment time.Duration) *Stitcher {
	return &Stitcher{
		randomSource: randomSource,
		baseTime:     time.Now(),
		counter:      0,
		increment:    increment,
	}
}

// Next returns the next stitched ULID from the sequence.
func (s *Stitcher) Next() ID {
	offset := time.Duration(s.counter) * s.increment
	s.counter++
	return StitchWithOffset(s.baseTime, offset, s.randomSource)
}

// Reset resets the sequence counter and refreshes the base time.
func (s *Stitcher) Reset() {
	s.counter = 0
	s.baseTime = time.Now()
}
