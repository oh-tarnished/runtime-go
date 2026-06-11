package ulid

import (
	"time"

	"github.com/oklog/ulid/v2"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ID wraps ulid.ULID and exposes a small, direct API for formatting,
// timestamp access, duration math, and protobuf conversion.
type ID struct {
	value ulid.ULID
}

// NewID creates an ID from an existing ulid.ULID.
func NewID(u ulid.ULID) ID {
	return ID{value: u}
}

// String returns the canonical 26-character ULID string.
func (id ID) String() string {
	return id.value.String()
}

// Time returns the ULID timestamp as a time.Time value.
func (id ID) Time() time.Time {
	return time.UnixMilli(int64(id.value.Time()))
}

// Date returns the calendar date portion of the ULID timestamp.
func (id ID) Date() time.Time {
	ts := id.Time()
	return time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, ts.Location())
}

// TimeCode returns the 48-bit timestamp portion of the ULID.
func (id ID) TimeCode() string {
	return id.value.String()[:10]
}

// GetTimeCode returns TimeCode and exists for backward compatibility.
func (id ID) GetTimeCode() string {
	return id.TimeCode()
}

// RandomCode returns the 80-bit random portion of the ULID.
func (id ID) RandomCode() string {
	return id.value.String()[10:]
}

// GetRandomCode returns RandomCode and exists for backward compatibility.
func (id ID) GetRandomCode() string {
	return id.RandomCode()
}

// MarshalText implements encoding.TextMarshaler.
func (id ID) MarshalText() ([]byte, error) {
	return id.value.MarshalText()
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (id *ID) UnmarshalText(data []byte) error {
	return id.value.UnmarshalText(data)
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (id ID) MarshalBinary() ([]byte, error) {
	return id.value.MarshalBinary()
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (id *ID) UnmarshalBinary(data []byte) error {
	return id.value.UnmarshalBinary(data)
}

// TimestampProto returns the timestamp as a protobuf Timestamp.
func (id ID) TimestampProto() *timestamppb.Timestamp {
	return timestamppb.New(id.Time())
}

// Age returns the duration since the ULID timestamp.
func (id ID) Age() time.Duration {
	return time.Since(id.Time())
}

// AgeProto returns Age as a protobuf Duration.
func (id ID) AgeProto() *durationpb.Duration {
	return durationpb.New(id.Age())
}

// To returns the duration between this ULID and another ULID.
func (id ID) To(other ID) time.Duration {
	return other.Time().Sub(id.Time())
}

// ToProto returns To as a protobuf Duration.
func (id ID) ToProto(other ID) *durationpb.Duration {
	return durationpb.New(id.To(other))
}

// Since returns the duration between the ULID timestamp and a target time.
func (id ID) Since(t time.Time) time.Duration {
	return t.Sub(id.Time())
}

// SinceProto returns Since as a protobuf Duration.
func (id ID) SinceProto(t time.Time) *durationpb.Duration {
	return durationpb.New(id.Since(t))
}
