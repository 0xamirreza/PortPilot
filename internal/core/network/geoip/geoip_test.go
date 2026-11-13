package geoip

import (
	"net/netip"
	"testing"
)

func TestLookup(t *testing.T) {
	tests := []struct {
		ip       string
		wantCode string
		wantFlag string
		isLocal  bool
	}{
		{"127.0.0.1", "HOST", "🏠", true},
		{"192.168.1.1", "LAN", "🔒", true},
		{"10.0.85.1", "LAN", "🔒", true},
		{"172.17.0.1", "LAN", "🔒", true},
		{"148.251.160.251", "DE", "🇩🇪", false},
		{"65.109.34.234", "FI", "🇫🇮", false},
		{"142.54.178.211", "US", "🇺🇸", false},
		{"5.202.75.165", "IR", "🇮🇷", false},
		{"116.203.91.91", "DE", "🇩🇪", false},
		{"8.8.8.8", "US", "🇺🇸", false},
		{"1.1.1.1", "US", "🇺🇸", false},
		{"149.154.167.99", "NL", "🇳🇱", false},
		{"188.114.99.1", "DE", "🇩🇪", false},
		{"2.1.1.1", "FR", "🇫🇷", false},
		{"2.200.1.1", "DE", "🇩🇪", false},
		{"62.70.1.1", "DE", "🇩🇪", false},
		{"185.10.1.1", "NL", "🇳🇱", false},
	}

	for _, tt := range tests {
		addr := netip.MustParseAddr(tt.ip)
		loc := Lookup(addr)
		if loc.CountryCode != tt.wantCode {
			t.Errorf("Lookup(%s).CountryCode = %q, want %q", tt.ip, loc.CountryCode, tt.wantCode)
		}
		if loc.Flag != tt.wantFlag {
			t.Errorf("Lookup(%s).Flag = %q, want %q", tt.ip, loc.Flag, tt.wantFlag)
		}
		if loc.IsLocal != tt.isLocal {
			t.Errorf("Lookup(%s).IsLocal = %v, want %v", tt.ip, loc.IsLocal, tt.isLocal)
		}
	}
}

