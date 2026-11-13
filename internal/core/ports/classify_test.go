package ports

import (
	"net/netip"
	"strings"
	"testing"
)

func TestClassifyExposure(t *testing.T) {
	cases := []struct {
		addr string
		want string
	}{
		{"127.0.0.1", ExposureLocalhost},
		{"127.1.2.3", ExposureLocalhost},
		{"0.0.0.0", ExposureWildcard},
		{"::", ExposureWildcard},
		{"::1", ExposureLocalhost},
		{"::ffff:127.0.0.1", ExposureLocalhost},
		{"::ffff:0.0.0.0", ExposureWildcard},
		{"192.168.1.10", ExposureSpecific},
		{"2001:db8::1", ExposureSpecific},
	}
	for _, tc := range cases {
		ip, err := netip.ParseAddr(tc.addr)
		if err != nil {
			t.Fatal(err)
		}
		if got := ClassifyExposure(ip); got != tc.want {
			t.Errorf("%s -> %s, want %s", tc.addr, got, tc.want)
		}
	}
}

func TestReachabilityDoesNotClaimInternet(t *testing.T) {
	text := strings.ToLower(Reachability(Socket{Exposure: ExposureWildcard}))
	for _, banned := range []string{"internet", "dangerous"} {
		if strings.Contains(text, banned) {
			t.Fatalf("wording contains %q: %s", banned, text)
		}
	}
}
