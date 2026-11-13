package geoip

import "testing"

func TestCoordinates(t *testing.T) {
	tests := []struct {
		code    string
		wantLat float64
		wantLon float64
	}{
		{"IR", 35.6892, 51.3890},
		{"US", 38.8951, -77.0364},
		{"DE", 50.1109, 8.6821},
		{"NL", 52.3676, 4.9041},
		{"FI", 60.1699, 24.9384},
		{"LOCAL", 35.6892, 51.3890},
		{"UNKNOWN_XYZ", 25.0, 0.0},
	}

	for _, tt := range tests {
		lat, lon := Coordinates(tt.code)
		if lat != tt.wantLat || lon != tt.wantLon {
			t.Errorf("Coordinates(%q) = (%v, %v), want (%v, %v)", tt.code, lat, lon, tt.wantLat, tt.wantLon)
		}
	}
}

