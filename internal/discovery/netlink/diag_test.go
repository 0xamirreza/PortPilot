package netlink

import (
	"encoding/binary"
	"testing"

	"github.com/0xamirreza/portpilot/internal/core/ports"
)

func TestParseDiagReadsTCPBytes(t *testing.T) {
	payload := make([]byte, 72+4+tcpBytesSentOff+8)
	payload[1] = 1
	binary.BigEndian.PutUint16(payload[4:6], 40000)
	binary.BigEndian.PutUint16(payload[6:8], 443)
	payload[8] = 10
	payload[24] = 1
	payload[27] = 1
	binary.LittleEndian.PutUint16(payload[72:74], uint16(4+tcpBytesSentOff+8))
	binary.LittleEndian.PutUint16(payload[74:76], inetDiagInfo)
	info := payload[76:]
	binary.LittleEndian.PutUint64(info[tcpBytesReceivedOff:tcpBytesReceivedOff+8], 50)
	binary.LittleEndian.PutUint64(info[tcpBytesSentOff:tcpBytesSentOff+8], 4096)

	sock, ok := parseDiag(payload, ports.IPv4, ports.TCP)
	if !ok || !sock.BytesKnown || sock.BytesSent != 4096 || sock.BytesReceived != 50 {
		t.Fatalf("ok=%v sock=%+v", ok, sock)
	}
	if sock.LocalPort != 40000 || sock.RemotePort != 443 || sock.State != "ESTABLISHED" {
		t.Fatalf("identity %+v", sock)
	}
}

func TestParseDiagIgnoresShortTCPInfo(t *testing.T) {
	payload := make([]byte, 72+4+16)
	payload[1] = 1
	binary.LittleEndian.PutUint16(payload[72:74], 20)
	binary.LittleEndian.PutUint16(payload[74:76], inetDiagInfo)
	sock, ok := parseDiag(payload, ports.IPv4, ports.TCP)
	if !ok || sock.BytesKnown {
		t.Fatalf("ok=%v known=%v", ok, sock.BytesKnown)
	}
}

func TestParseDiagSkipsBytesForUDP(t *testing.T) {
	payload := make([]byte, 72+4+tcpBytesSentOff+8)
	payload[1] = 1
	binary.LittleEndian.PutUint16(payload[72:74], uint16(4+tcpBytesSentOff+8))
	binary.LittleEndian.PutUint16(payload[74:76], inetDiagInfo)
	payload[76+tcpBytesSentOff] = 9
	sock, ok := parseDiag(payload, ports.IPv4, ports.UDP)
	if !ok || sock.BytesKnown || sock.BytesSent != 0 {
		t.Fatalf("ok=%v sock=%+v", ok, sock)
	}
}
