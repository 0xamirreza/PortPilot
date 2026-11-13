package ports

import (
	"net/netip"
	"testing"
)

func TestTopologyGroupsPeersAndSkipsListeners(t *testing.T) {
	listen := Socket{Protocol: TCP, Family: IPv4, LocalIP: netip.MustParseAddr("0.0.0.0"), LocalPort: 22, State: "LISTEN", Process: "sshd", PID: 1}
	udp := Socket{Protocol: UDP, Family: IPv4, LocalIP: netip.MustParseAddr("0.0.0.0"), LocalPort: 53, State: "UNCONN", Process: "dns", PID: 2}
	a := Socket{
		Protocol: TCP, Family: IPv4, LocalIP: netip.MustParseAddr("10.0.0.2"), LocalPort: 40000,
		RemoteIP: netip.MustParseAddr("1.1.1.1"), RemotePort: 443, State: "ESTABLISHED",
		Process: "firefox", PID: 9, BytesSent: 100, BytesReceived: 40, BytesKnown: true,
	}
	b := Socket{
		Protocol: TCP, Family: IPv4, LocalIP: netip.MustParseAddr("10.0.0.2"), LocalPort: 40000,
		RemoteIP: netip.MustParseAddr("127.0.0.1"), RemotePort: 8080, State: "ESTABLISHED",
		Process: "firefox", PID: 9, BytesSent: 500, BytesReceived: 5, BytesKnown: true,
	}
	hubs := Topology([]Socket{listen, udp, a, b})
	if len(hubs) != 1 {
		t.Fatalf("hubs %d", len(hubs))
	}
	h := hubs[0]
	if h.Title != "firefox" || h.Sent != 600 || h.Recv != 45 || len(h.Peers) != 2 {
		t.Fatalf("%+v", h)
	}
	if h.PID != 9 || h.Process != "firefox" || h.Protocol != TCP || h.LocalPort != 40000 {
		t.Fatalf("hub fields %+v", h)
	}
	if len(h.Sockets) != 2 {
		t.Fatalf("hub sockets count: %d", len(h.Sockets))
	}
	if h.Peers[0].Remote != "127.0.0.1:8080" || !h.Peers[0].OnHost {
		t.Fatalf("order %+v", h.Peers)
	}
	if h.Peers[1].OnHost {
		t.Fatal("public peer marked on host")
	}
}

func TestTopologyLeavesUnknownBytesUnset(t *testing.T) {
	s := Socket{
		Protocol: UDP, Family: IPv4, LocalIP: netip.MustParseAddr("10.0.0.2"), LocalPort: 1,
		RemoteIP: netip.MustParseAddr("10.1.1.1"), RemotePort: 53, State: "ESTABLISHED",
		Process: "resolver", PID: 3,
	}
	hubs := Topology([]Socket{s})
	if len(hubs) != 1 || hubs[0].Known || hubs[0].Peers[0].BytesKnown {
		t.Fatalf("%+v", hubs)
	}
}

func TestTopologyAggregatesClosingSockets(t *testing.T) {
	c1 := Socket{
		Protocol: TCP, Family: IPv4, LocalIP: netip.MustParseAddr("192.168.1.10"), LocalPort: 45996,
		RemoteIP: netip.MustParseAddr("142.251.156.7"), RemotePort: 443, State: "FIN-WAIT-1",
	}
	c2 := Socket{
		Protocol: TCP, Family: IPv4, LocalIP: netip.MustParseAddr("192.168.1.10"), LocalPort: 38840,
		RemoteIP: netip.MustParseAddr("216.239.38.120"), RemotePort: 443, State: "FIN-WAIT-1",
	}
	hubs := Topology([]Socket{c1, c2})
	if len(hubs) != 1 {
		t.Fatalf("expected 1 aggregated hub for closing sockets, got %d", len(hubs))
	}
	if hubs[0].Title != "kernel (closing sockets)" {
		t.Fatalf("expected 'kernel (closing sockets)', got %q", hubs[0].Title)
	}
	if len(hubs[0].Peers) != 2 {
		t.Fatalf("expected 2 peers in closing hub, got %d", len(hubs[0].Peers))
	}
}
