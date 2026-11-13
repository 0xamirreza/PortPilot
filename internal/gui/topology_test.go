package gui

import (
	"net/netip"
	"testing"
	"time"

	"github.com/0xamirreza/portpilot/internal/core/ports"
)

func TestFormatRTT(t *testing.T) {
	if got := formatRTT(0); got != "-" {
		t.Fatalf("expected -, got %s", got)
	}
	if got := formatRTT(450); got != "450 µs" {
		t.Fatalf("expected 450 µs, got %s", got)
	}
	if got := formatRTT(15400); got != "15.4 ms" {
		t.Fatalf("expected 15.4 ms, got %s", got)
	}
}

func TestFormatOctets(t *testing.T) {
	if got := formatOctets(500); got != "500 B" {
		t.Fatalf("got %s", got)
	}
	if got := formatOctets(2048); got != "2.0 KB" {
		t.Fatalf("got %s", got)
	}
	if got := formatOctets(1048576 * 5); got != "5.0 MB" {
		t.Fatalf("got %s", got)
	}
}

func TestLiveStreamTracking(t *testing.T) {
	ui := &UI{}
	sock1 := ports.Socket{
		Protocol:      ports.TCP,
		LocalIP:       netip.MustParseAddr("127.0.0.1"),
		LocalPort:     40000,
		RemoteIP:      netip.MustParseAddr("1.1.1.1"),
		RemotePort:    443,
		PID:           100,
		Process:       "testproc",
		BytesSent:     1000,
		BytesReceived: 2000,
		SegsOut:       10,
		SegsIn:        20,
		State:         "ESTABLISHED",
		BytesKnown:    true,
	}

	h := ports.Hub{
		Key:       ports.HubKey(sock1),
		Protocol:  ports.TCP,
		LocalPort: 40000,
		PID:       100,
		Process:   "testproc",
	}
	ui.openLiveSource(h)
	if ui.topoSelectedHubKey != h.Key {
		t.Fatalf("expected key %s", h.Key)
	}

	var mockSocks []ports.Socket
	oldDump := dumpSockets
	defer func() { dumpSockets = oldDump }()
	dumpSockets = func() ([]ports.Socket, error) {
		return mockSocks, nil
	}

	mockSocks = []ports.Socket{sock1}
	// 1st tick establishes baseline
	ui.updateLiveStream(h, mockSocks)
	if len(ui.livePackets) != 1 || (ui.livePackets[0].Direction != "INIT" && ui.livePackets[0].Direction != "ACTIVE") {
		t.Fatalf("expected 1 baseline packet, got %+v", ui.livePackets)
	}

	// Fast forward time for second tick
	ui.liveLastSample = ui.liveLastSample.Add(-300 * time.Millisecond)

	// 2nd tick with new traffic
	sock2 := sock1
	sock2.BytesSent += 1400
	sock2.BytesReceived += 2800
	sock2.SegsOut += 1
	sock2.SegsIn += 2

	mockSocks = []ports.Socket{sock2}
	ui.updateLiveStream(h, mockSocks)
	if len(ui.livePackets) < 3 {
		t.Fatalf("expected at least 3 events, got %d: %+v", len(ui.livePackets), ui.livePackets)
	}

	hasTX, hasRX := false, false
	for _, ev := range ui.livePackets[:2] {
		if ev.Direction == "TX" && ev.Bytes == 1400 {
			hasTX = true
		}
		if ev.Direction == "RX" && ev.Bytes == 2800 {
			hasRX = true
		}
	}
	if !hasTX || !hasRX {
		t.Fatalf("expected TX and RX events, got: %+v", ui.livePackets[:2])
	}
}
