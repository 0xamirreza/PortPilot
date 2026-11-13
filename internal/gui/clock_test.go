package gui

import (
	"testing"
	"time"
)

func TestScanClockReportsLive(t *testing.T) {
	start := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	if got := scanClock(start, time.Time{}, start, false); got != "STARTING" {
		t.Fatalf("expected STARTING, got %s", got)
	}
	if got := scanClock(start, start, start, true); got != "● LIVE" {
		t.Fatalf("expected ● LIVE, got %s", got)
	}
	if got := scanClock(start, start, start, false); got != "● LIVE" {
		t.Fatalf("expected ● LIVE, got %s", got)
	}
}
