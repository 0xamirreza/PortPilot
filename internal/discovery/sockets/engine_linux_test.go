//go:build linux

package sockets

import (
	"net"
	"testing"
	"time"

	"github.com/0xamirreza/portpilot/internal/core/ports"
)

func TestScanFindsLocalListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := uint16(ln.Addr().(*net.TCPAddr).Port)

	snap, err := (&Engine{}).Scan()
	if err != nil {
		t.Fatal(err)
	}
	if snap.Source != "netlink" && snap.Source != "procfs" {
		t.Fatalf("source %q", snap.Source)
	}
	var found *ports.Socket
	for i := range snap.Sockets {
		s := &snap.Sockets[i]
		if s.Protocol == ports.TCP && s.LocalPort == port && s.State == "LISTEN" {
			found = s
			break
		}
	}
	if found == nil {
		t.Fatalf("listener on %d not found via %s (%d sockets)", port, snap.Source, len(snap.Sockets))
	}
	if found.Exposure != ports.ExposureLocalhost {
		t.Fatalf("exposure %s", found.Exposure)
	}
	if !found.OwnerKnown || found.PID != osGetpid() {
		t.Fatalf("owner pid=%d known=%v", found.PID, found.OwnerKnown)
	}
	if snap.Duration <= 0 || snap.ScannedAt.IsZero() {
		t.Fatal("scan metadata")
	}
	if len(snap.Interfaces) == 0 {
		t.Fatal("no interfaces")
	}
}

func TestSecondScanCanReportCPU(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	eng := &Engine{}
	if _, err := eng.Scan(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	snap, err := eng.Scan()
	if err != nil {
		t.Fatal(err)
	}
	pid := osGetpid()
	for _, p := range snap.Processes {
		if p.PID == pid {
			if p.StartTicks == 0 {
				t.Fatal("missing start ticks")
			}
			return
		}
	}
	t.Fatal("test process missing from second scan")
}

func TestEngineWithPrivilegedProcs(t *testing.T) {
	eng := &Engine{}
	eng.SetPrivilegedProcs(nil)
	if len(eng.privProcs) != 0 {
		t.Fatalf("expected 0 privProcs")
	}
}
