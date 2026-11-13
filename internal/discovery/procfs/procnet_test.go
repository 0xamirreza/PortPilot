package procfs

import (
	"strings"
	"testing"

	"github.com/0xamirreza/portpilot/internal/core/ports"
)

const tcpFixture = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 12345 1 0000000000000000 100 0 0 10 0
   1: 00000000:0016 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 55 1 0000000000000000 100 0 0 10 0
`

const tcp6Fixture = `  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000000000000000000000000000:0050 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 9 1 0000000000000000 100 0 0 10 0
   1: 00000000000000000000000001000000:01BB 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 8 1 0000000000000000 100 0 0 10 0
`

func TestParseProcNetTCP(t *testing.T) {
	got, err := ParseProcNet(strings.NewReader(tcpFixture), ports.IPv4, ports.TCP)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].LocalIP.String() != "127.0.0.1" || got[0].LocalPort != 8080 || got[0].State != "LISTEN" || got[0].UID != 1000 || got[0].Inode != 12345 {
		t.Fatalf("first %+v", got[0])
	}
	if got[0].Exposure != ports.ExposureLocalhost {
		t.Fatal(got[0].Exposure)
	}
	if got[1].LocalIP.String() != "0.0.0.0" || got[1].LocalPort != 22 || got[1].Exposure != ports.ExposureWildcard {
		t.Fatalf("second %+v", got[1])
	}
}

func TestParseProcNetTCP6(t *testing.T) {
	got, err := ParseProcNet(strings.NewReader(tcp6Fixture), ports.IPv6, ports.TCP)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].LocalIP.String() != "::" || got[0].LocalPort != 80 || got[0].Exposure != ports.ExposureWildcard {
		t.Fatalf("wildcard %+v", got[0])
	}
	if got[1].LocalIP.String() != "::1" || got[1].LocalPort != 443 || got[1].Exposure != ports.ExposureLocalhost {
		t.Fatalf("loopback %+v", got[1])
	}
}

func TestUDPUnconnectedState(t *testing.T) {
	line := "0: 00000000:0035 00000000:0000 07 00000000:00000000 00:00000000 00000000 0 0 42 1 0 0 0 0"
	got, err := ParseProcNet(strings.NewReader(line), ports.IPv4, ports.UDP)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].State != "UNCONN" || got[0].LocalPort != 53 {
		t.Fatalf("%+v", got[0])
	}
	if !got[0].IsBoundService() {
		t.Fatal("udp should count as a bound socket")
	}
}
