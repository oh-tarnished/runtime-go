package ulid

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGenerateAndComponents(t *testing.T) {
	id := Generate()

	if len(id.String()) != 26 {
		t.Fatalf("expected 26-char ULID, got %q", id.String())
	}

	if len(id.TimeCode()) != 10 {
		t.Fatalf("expected 10-char time code, got %q", id.TimeCode())
	}

	if len(id.RandomCode()) != 16 {
		t.Fatalf("expected 16-char random code, got %q", id.RandomCode())
	}

	if id.TimeCode()+id.RandomCode() != id.String() {
		t.Fatalf("time and random parts should reconstruct the ULID")
	}
}

func TestTimeAndProtoHelpers(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	id, err := FromTime(now)
	if err != nil {
		t.Fatalf("FromTime failed: %v", err)
	}

	if diff := id.Time().Sub(now); diff < -time.Millisecond || diff > time.Millisecond {
		t.Fatalf("unexpected time delta: %v", diff)
	}

	if !id.TimestampProto().AsTime().Equal(id.Time()) {
		t.Fatalf("TimestampProto should round-trip the timestamp")
	}

	if got := id.Date(); got.Day() != now.Day() || got.Month() != now.Month() || got.Year() != now.Year() {
		t.Fatalf("Date should preserve calendar day, got %v want %v", got, now)
	}
}

func TestDurationAndStitch(t *testing.T) {
	base := time.Now().UTC().Truncate(time.Millisecond)
	older, err := FromTime(base.Add(-100 * time.Millisecond))
	if err != nil {
		t.Fatalf("FromTime failed: %v", err)
	}
	newer, err := FromTime(base)
	if err != nil {
		t.Fatalf("FromTime failed: %v", err)
	}

	if got := older.To(newer); got < 90*time.Millisecond || got > 110*time.Millisecond {
		t.Fatalf("unexpected duration: %v", got)
	}

	stitched := Stitch(newer, older)
	if stitched.TimeCode() != newer.TimeCode() {
		t.Fatalf("stitched timestamp mismatch")
	}
	if stitched.RandomCode() != older.RandomCode() {
		t.Fatalf("stitched random mismatch")
	}

	if got := older.AgeProto(); got == nil || got.AsDuration() < 0 {
		t.Fatalf("AgeProto should return a non-nil duration")
	}

	if got := newer.Since(base.Add(100 * time.Millisecond)); got < 90*time.Millisecond || got > 110*time.Millisecond {
		t.Fatalf("unexpected since duration: %v", got)
	}

	if got := newer.SinceProto(base.Add(100 * time.Millisecond)); got == nil || got.AsDuration() < 90*time.Millisecond {
		t.Fatalf("unexpected since proto duration")
	}

	if got := older.ToProto(newer); got == nil || got.AsDuration() < 90*time.Millisecond {
		t.Fatalf("unexpected to proto duration")
	}
}

func TestParseAndMarshalRoundTrip(t *testing.T) {
	original := Generate()

	parsed, err := Parse(original.String())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if parsed.String() != original.String() {
		t.Fatalf("Parse round trip mismatch")
	}

	validated, err := Validate(original.String())
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if validated.String() != original.String() {
		t.Fatalf("Validate round trip mismatch")
	}

	data, err := original.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText failed: %v", err)
	}
	var textRoundTrip ID
	if err = textRoundTrip.UnmarshalText(data); err != nil {
		t.Fatalf("UnmarshalText failed: %v", err)
	}
	if textRoundTrip.String() != original.String() {
		t.Fatalf("text round trip mismatch")
	}

	bin, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}
	var binaryRoundTrip ID
	if err := binaryRoundTrip.UnmarshalBinary(bin); err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}
	if binaryRoundTrip.String() != original.String() {
		t.Fatalf("binary round trip mismatch")
	}

	if _, err := FromTimestampPB(timestamppb.New(time.Now())); err != nil {
		t.Fatalf("FromTimestampPB failed: %v", err)
	}
	if _, err := FromDurationPB(durationpb.New(5 * time.Second)); err != nil {
		t.Fatalf("FromDurationPB failed: %v", err)
	}
}
