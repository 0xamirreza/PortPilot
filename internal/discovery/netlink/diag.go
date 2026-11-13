// Package netlink reads socket tables through SOCK_DIAG.
package netlink

import (
	"encoding/binary"
	"fmt"
	"net/netip"
	"time"

	"github.com/0xamirreza/portpilot/internal/core/ports"

	"golang.org/x/sys/unix"
)

// DumpSockets returns TCP and UDP sockets for IPv4 and IPv6.
// The caller falls back to procfs when this returns an error.
func DumpSockets() ([]ports.Socket, error) {
	var out []ports.Socket
	queries := []struct {
		family    uint8
		proto     uint8
		famName   string
		protoName string
	}{
		{unix.AF_INET, unix.IPPROTO_TCP, ports.IPv4, ports.TCP},
		{unix.AF_INET6, unix.IPPROTO_TCP, ports.IPv6, ports.TCP},
		{unix.AF_INET, unix.IPPROTO_UDP, ports.IPv4, ports.UDP},
		{unix.AF_INET6, unix.IPPROTO_UDP, ports.IPv6, ports.UDP},
	}
	for _, q := range queries {
		batch, err := dumpFamily(q.family, q.proto, q.famName, q.protoName)
		if err != nil {
			return nil, err
		}
		out = append(out, batch...)
	}
	return out, nil
}

func dumpFamily(family, proto uint8, famName, protoName string) ([]ports.Socket, error) {
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW|unix.SOCK_CLOEXEC, unix.NETLINK_INET_DIAG)
	if err != nil {
		return nil, fmt.Errorf("sock_diag socket: %w", err)
	}
	defer unix.Close(fd)
	if err := unix.Bind(fd, &unix.SockaddrNetlink{Family: unix.AF_NETLINK}); err != nil {
		return nil, fmt.Errorf("bind sock_diag: %w", err)
	}
	tv := unix.NsecToTimeval((3 * time.Second).Nanoseconds())
	if err := unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &tv); err != nil {
		return nil, err
	}
	req := make([]byte, 16+56)
	binary.LittleEndian.PutUint32(req[0:4], uint32(len(req)))
	binary.LittleEndian.PutUint16(req[4:6], uint16(unix.SOCK_DIAG_BY_FAMILY))
	binary.LittleEndian.PutUint16(req[6:8], unix.NLM_F_REQUEST|unix.NLM_F_DUMP)
	binary.LittleEndian.PutUint32(req[8:12], 1)
	req[16] = family
	req[17] = proto
	// idiag_ext bit (INET_DIAG_INFO-1) asks for struct tcp_info after the diag message.
	req[18] = 1 << (inetDiagInfo - 1)
	binary.LittleEndian.PutUint32(req[20:24], 0xffffffff)
	if err := unix.Sendto(fd, req, 0, &unix.SockaddrNetlink{Family: unix.AF_NETLINK}); err != nil {
		return nil, fmt.Errorf("send sock_diag: %w", err)
	}
	buf := make([]byte, 1<<16)
	var out []ports.Socket
	for {
		n, _, err := unix.Recvfrom(fd, buf, 0)
		if err != nil {
			return nil, fmt.Errorf("recv sock_diag: %w", err)
		}
		done, batch, err := parseDump(buf[:n], famName, protoName)
		if err != nil {
			return nil, err
		}
		out = append(out, batch...)
		if done {
			return out, nil
		}
	}
}

func parseDump(buf []byte, family, proto string) (bool, []ports.Socket, error) {
	var out []ports.Socket
	for len(buf) >= 16 {
		mlen := int(binary.LittleEndian.Uint32(buf[0:4]))
		if mlen < 16 || mlen > len(buf) {
			return false, nil, fmt.Errorf("truncated netlink message")
		}
		mtype := binary.LittleEndian.Uint16(buf[4:6])
		payload := buf[16:mlen]
		switch mtype {
		case unix.NLMSG_DONE:
			return true, out, nil
		case unix.NLMSG_ERROR:
			if len(payload) < 4 {
				return false, nil, fmt.Errorf("short netlink error")
			}
			errno := int32(binary.LittleEndian.Uint32(payload[:4]))
			if errno == 0 {
				return true, out, nil
			}
			return false, nil, fmt.Errorf("sock_diag: %w", unix.Errno(-errno))
		case unix.SOCK_DIAG_BY_FAMILY:
			if sock, ok := parseDiag(payload, family, proto); ok {
				out = append(out, sock)
			}
		}
		buf = buf[align(mlen):]
	}
	return false, out, nil
}

func parseDiag(payload []byte, family, proto string) (ports.Socket, bool) {
	if len(payload) < 72 {
		return ports.Socket{}, false
	}
	local := readAddr(payload[8:24], family)
	remote := readAddr(payload[24:40], family)
	state := payload[1]
	sock := ports.Socket{
		Protocol:   proto,
		Family:     family,
		LocalIP:    local,
		LocalPort:  binary.BigEndian.Uint16(payload[4:6]),
		RemoteIP:   remote,
		RemotePort: binary.BigEndian.Uint16(payload[6:8]),
		State:      diagState(proto, state),
		IfIndex:    binary.LittleEndian.Uint32(payload[40:44]),
		UID:        binary.LittleEndian.Uint32(payload[64:68]),
		Inode:      uint64(binary.LittleEndian.Uint32(payload[68:72])),
		Exposure:   ports.ClassifyExposure(local),
		Source:     "netlink",
	}
	readTCPBytes(&sock, payload[72:])
	return sock, true
}

// inetDiagInfo is the uapi attribute that carries struct tcp_info.
const inetDiagInfo = 2

// Offsets inside struct tcp_info on little-endian Linux, after the leading
// 8-byte flag block and 24 uint32 fields. bytes_sent sits past the padding
// the compiler inserts before each uint64 that follows a uint32.
const (
	tcpRTTMicrosecOff   = 68
	tcpBytesReceivedOff = 128
	tcpSegsOutOff       = 136
	tcpSegsInOff        = 140
	tcpBytesSentOff     = 200
)

func readTCPBytes(sock *ports.Socket, attrs []byte) {
	if sock.Protocol != ports.TCP {
		return
	}
	for len(attrs) >= 4 {
		nlaLen := int(binary.LittleEndian.Uint16(attrs[0:2]))
		nlaType := binary.LittleEndian.Uint16(attrs[2:4]) & 0x3fff
		if nlaLen < 4 || nlaLen > len(attrs) {
			return
		}
		if nlaType == inetDiagInfo {
			info := attrs[4:nlaLen]
			if len(info) < tcpBytesSentOff+8 {
				return
			}
			sock.BytesReceived = binary.LittleEndian.Uint64(info[tcpBytesReceivedOff : tcpBytesReceivedOff+8])
			sock.BytesSent = binary.LittleEndian.Uint64(info[tcpBytesSentOff : tcpBytesSentOff+8])
			sock.SegsOut = binary.LittleEndian.Uint32(info[tcpSegsOutOff : tcpSegsOutOff+4])
			sock.SegsIn = binary.LittleEndian.Uint32(info[tcpSegsInOff : tcpSegsInOff+4])
			sock.RTTMicrosec = binary.LittleEndian.Uint32(info[tcpRTTMicrosecOff : tcpRTTMicrosecOff+4])
			sock.BytesKnown = true
			return
		}
		attrs = attrs[(nlaLen+3)&^3:]
	}
}

func readAddr(b []byte, family string) netip.Addr {
	if family == ports.IPv4 {
		var v4 [4]byte
		copy(v4[:], b[:4])
		return netip.AddrFrom4(v4)
	}
	var v6 [16]byte
	copy(v6[:], b[:16])
	return netip.AddrFrom16(v6)
}

func diagState(proto string, code uint8) string {
	if proto == ports.UDP {
		switch code {
		case 1:
			return "ESTABLISHED"
		case 7:
			return "UNCONN"
		default:
			return "UDP"
		}
	}
	names := map[uint8]string{
		1: "ESTABLISHED", 2: "SYN-SENT", 3: "SYN-RECV", 4: "FIN-WAIT-1", 5: "FIN-WAIT-2",
		6: "TIME-WAIT", 7: "CLOSE", 8: "CLOSE-WAIT", 9: "LAST-ACK", 10: "LISTEN", 11: "CLOSING",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return "TCP"
}

func align(n int) int {
	return (n + 3) &^ 3
}
