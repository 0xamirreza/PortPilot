package procfs

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net/netip"
	"strconv"
	"strings"

	"github.com/0xamirreza/portpilot/internal/core/ports"
)

// ParseProcNet reads /proc/net/tcp, tcp6, udp, or udp6.
func ParseProcNet(r io.Reader, family, proto string) ([]ports.Socket, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var out []ports.Socket
	first := true
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if first {
			first = false
			if strings.Contains(line, "local_address") {
				continue
			}
		}
		sock, err := parseProcLine(line, family, proto)
		if err != nil {
			return nil, err
		}
		out = append(out, sock)
	}
	return out, sc.Err()
}

func parseProcLine(line, family, proto string) (ports.Socket, error) {
	fields := strings.Fields(line)
	if len(fields) < 10 {
		return ports.Socket{}, fmt.Errorf("short proc net line: %q", line)
	}
	localIP, localPort, err := parseHexEndpoint(fields[1], family)
	if err != nil {
		return ports.Socket{}, fmt.Errorf("local address: %w", err)
	}
	remoteIP, remotePort, err := parseHexEndpoint(fields[2], family)
	if err != nil {
		return ports.Socket{}, fmt.Errorf("remote address: %w", err)
	}
	stateCode, err := strconv.ParseUint(fields[3], 16, 8)
	if err != nil {
		return ports.Socket{}, fmt.Errorf("state: %w", err)
	}
	uid, err := strconv.ParseUint(fields[7], 10, 32)
	if err != nil {
		return ports.Socket{}, fmt.Errorf("uid: %w", err)
	}
	inode, err := strconv.ParseUint(fields[9], 10, 64)
	if err != nil {
		return ports.Socket{}, fmt.Errorf("inode: %w", err)
	}
	sock := ports.Socket{
		Protocol:   proto,
		Family:     family,
		LocalIP:    localIP,
		LocalPort:  localPort,
		RemoteIP:   remoteIP,
		RemotePort: remotePort,
		State:      stateName(proto, uint8(stateCode)),
		UID:        uint32(uid),
		Inode:      inode,
		Exposure:   ports.ClassifyExposure(localIP),
		Source:     "procfs",
	}
	return sock, nil
}

func parseHexEndpoint(field, family string) (netip.Addr, uint16, error) {
	addr, portHex, ok := strings.Cut(field, ":")
	if !ok {
		return netip.Addr{}, 0, fmt.Errorf("missing port in %q", field)
	}
	portVal, err := strconv.ParseUint(portHex, 16, 16)
	if err != nil {
		return netip.Addr{}, 0, err
	}
	var ip netip.Addr
	switch family {
	case ports.IPv4:
		ip, err = parseHexIPv4(addr)
	case ports.IPv6:
		ip, err = parseHexIPv6(addr)
	default:
		err = fmt.Errorf("family %s", family)
	}
	if err != nil {
		return netip.Addr{}, 0, err
	}
	return ip, uint16(portVal), nil
}

func parseHexIPv4(s string) (netip.Addr, error) {
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return netip.Addr{}, err
	}
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], uint32(v))
	return netip.AddrFrom4(b), nil
}

func parseHexIPv6(s string) (netip.Addr, error) {
	if len(s) != 32 {
		return netip.Addr{}, fmt.Errorf("ipv6 hex length %d", len(s))
	}
	var b [16]byte
	for i := 0; i < 4; i++ {
		word, err := strconv.ParseUint(s[i*8:(i+1)*8], 16, 32)
		if err != nil {
			return netip.Addr{}, err
		}
		binary.LittleEndian.PutUint32(b[i*4:], uint32(word))
	}
	return netip.AddrFrom16(b), nil
}

func stateName(proto string, code uint8) string {
	if proto == ports.UDP {
		switch code {
		case 1:
			return "ESTABLISHED"
		case 7:
			return "UNCONN"
		default:
			return "UDP-" + strconv.Itoa(int(code))
		}
	}
	names := map[uint8]string{
		1:  "ESTABLISHED",
		2:  "SYN-SENT",
		3:  "SYN-RECV",
		4:  "FIN-WAIT-1",
		5:  "FIN-WAIT-2",
		6:  "TIME-WAIT",
		7:  "CLOSE",
		8:  "CLOSE-WAIT",
		9:  "LAST-ACK",
		10: "LISTEN",
		11: "CLOSING",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return "TCP-" + strconv.Itoa(int(code))
}
