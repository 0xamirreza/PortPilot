package ports

import (
	"net/netip"
	"testing"
)

func TestDefaultQueryHidesEstablishedTCP(t *testing.T) {
	q := DefaultQuery()
	listen := Socket{Protocol: TCP, Family: IPv4, LocalIP: netip.MustParseAddr("0.0.0.0"), LocalPort: 22, State: "LISTEN", Exposure: ExposureWildcard, Process: "sshd"}
	estab := Socket{Protocol: TCP, Family: IPv4, LocalIP: netip.MustParseAddr("10.0.0.2"), LocalPort: 443, State: "ESTABLISHED", Exposure: ExposureSpecific, Process: "curl"}
	udp := Socket{Protocol: UDP, Family: IPv6, LocalIP: netip.MustParseAddr("::"), LocalPort: 53, State: "UNCONN", Exposure: ExposureWildcard, Process: "dns"}
	got := Apply([]Socket{listen, estab, udp}, q)
	if len(got) != 2 {
		t.Fatalf("got %d sockets", len(got))
	}
	q.Active = true
	q.Listening = false
	got = Apply([]Socket{listen, estab, udp}, q)
	if len(got) != 1 || got[0].Process != "curl" {
		t.Fatalf("active filter = %+v", got)
	}
}

func TestQueryTextMatchesPortAndProcess(t *testing.T) {
	s := Socket{Protocol: TCP, Family: IPv4, LocalIP: netip.MustParseAddr("127.0.0.1"), LocalPort: 8080, State: "LISTEN", Exposure: ExposureLocalhost, Process: "PortPilot"}
	q := DefaultQuery()
	q.Text = "8080"
	if !q.Match(s) {
		t.Fatal("port")
	}
	q.Text = "pilot"
	if !q.Match(s) {
		t.Fatal("process")
	}
	q.Text = "sshd"
	if q.Match(s) {
		t.Fatal("unexpected match")
	}
}
