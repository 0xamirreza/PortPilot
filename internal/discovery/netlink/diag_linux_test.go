package netlink

import "testing"

func TestLiveEstablishedTCPHasByteCounters(t *testing.T) {
	socks, err := DumpSockets()
	if err != nil {
		t.Fatal(err)
	}
	established := 0
	known := 0
	for _, s := range socks {
		if s.Protocol == "TCP" && s.State == "ESTABLISHED" {
			established++
			if s.BytesKnown {
				known++
			}
		}
	}
	if established > 0 && known == 0 {
		t.Fatal("established TCP sockets had no kernel byte counters")
	}
}
