package timeutil

import (
	"strings"
	"testing"
	"time"
)

func TestNowIsBeijing(t *testing.T) {
	n := Now()
	_, off := n.Zone()
	if off != 8*3600 {
		t.Fatalf("offset %d", off)
	}
}

func TestFormatParse(t *testing.T) {
	ts := time.Date(2026, 8, 23, 12, 0, 0, 0, Beijing())
	s := Format(ts)
	if !strings.HasPrefix(s, "2026-08-23 12:00:00") {
		t.Fatalf("got %s", s)
	}
	back, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	if !back.Equal(ts) {
		t.Fatalf("%v != %v", back, ts)
	}
}

func TestDurationMS(t *testing.T) {
	if DurationMS(0) != 0 {
		t.Fatal("zero")
	}
	if DurationMS(500*time.Microsecond) != 1 {
		t.Fatal("sub-ms should clamp to 1")
	}
	if DurationMS(12*time.Millisecond) != 12 {
		t.Fatal("12ms")
	}
}
