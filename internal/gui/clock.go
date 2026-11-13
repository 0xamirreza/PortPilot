package gui

import (
	"time"
)

// scanClock reports live status in the header.
func scanClock(now, scanned, next time.Time, scanning bool) string {
	if scanned.IsZero() {
		return "STARTING"
	}
	return "● LIVE"
}
