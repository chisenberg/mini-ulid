package miniulid

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestGenerateWithComponents(t *testing.T) {
	ts := time.Date(2023, 5, 15, 12, 34, 56, 789, time.FixedZone("PDT?", -7*3600))
	id, err := GenerateWithComponents(ts, 0x1ACE)
	if err != nil {
		t.Fatalf("GenerateWithComponents error: %v", err)
	}

	days, minutes, random := id.Components()
	diffDays := int(ts.UTC().Sub(epoch) / (24 * time.Hour))
	if got, want := int(days), diffDays; got != want {
		t.Fatalf("days mismatch: got %d want %d", got, want)
	}
	if got, want := int(minutes), ts.UTC().Hour()*60+ts.UTC().Minute(); got != want {
		t.Fatalf("minutes mismatch: got %d want %d", got, want)
	}
	if got, want := int(random), int(0x1ACE&randomMask); got != want {
		t.Fatalf("random mismatch: got %d want %d", got, want)
	}

	encoded := id.String()
	if len(encoded) != totalSize {
		t.Fatalf("encoded length: got %d want %d", len(encoded), totalSize)
	}

	parsed, err := Parse(encoded)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if parsed != id {
		t.Fatalf("Parse mismatch: got %v want %v", parsed, id)
	}

	intValue := id.Int64()
	back, err := FromInt64(intValue)
	if err != nil {
		t.Fatalf("FromInt64 error: %v", err)
	}
	if back != id {
		t.Fatalf("FromInt64 mismatch: got %v want %v", back, id)
	}

	roundTripTime := id.Time()
	expectedTime := ts.UTC().Truncate(time.Minute)
	if !roundTripTime.Equal(expectedTime) {
		t.Fatalf("Time mismatch: got %v want %v", roundTripTime, expectedTime)
	}
}

func TestGenerateWithTimeEntropy(t *testing.T) {
	ts := time.Date(2024, 7, 30, 8, 0, 0, 0, time.UTC)
	entropy := bytes.NewReader([]byte{0x12, 0x34})
	id, err := GenerateWithTime(ts, entropy)
	if err != nil {
		t.Fatalf("GenerateWithTime error: %v", err)
	}

	_, _, random := id.Components()
	if random != uint16(0x1234)&randomMask {
		t.Fatalf("unexpected random: got %d", random)
	}

	got := id.Time()
	if want := ts; !got.Equal(want) {
		t.Fatalf("time mismatch: got %v want %v", got, want)
	}
}

func TestGenerateErrors(t *testing.T) {
	_, err := GenerateWithComponents(epoch.Add(-time.Minute), 0)
	if !errors.Is(err, errTimePast) {
		t.Fatalf("expected errTimePast, got %v", err)
	}

	_, err = GenerateWithComponents(epoch, randomMask+1)
	if err == nil {
		t.Fatalf("expected random overflow error")
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := Parse("ABC"); !errors.Is(err, errLength) {
		t.Fatalf("expected errLength, got %v", err)
	}
	if _, err := Parse("!!!!!!!!"); err == nil || !errors.Is(err, errInvalidChar) {
		t.Fatalf("expected errInvalidChar, got %v", err)
	}
	if _, err := FromInt64(1<<totalBits | 1); err == nil {
		t.Fatalf("expected overflow error")
	}
}
