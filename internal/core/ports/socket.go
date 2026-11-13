// Package ports models listening and bound sockets.
package ports

import "net/netip"

const (
	TCP = "TCP"
	UDP = "UDP"

	IPv4 = "IPv4"
	IPv6 = "IPv6"

	ExposureLocalhost = "localhost"
	ExposureWildcard  = "wildcard"
	ExposureSpecific  = "specific"
	ExposureUnknown   = "unknown"
)

// Socket is one observed transport endpoint.
// A wildcard bind is not evidence that the port is reachable from another host.
type Socket struct {
	Protocol      string
	Family        string
	LocalIP       netip.Addr
	LocalPort     uint16
	RemoteIP      netip.Addr
	RemotePort    uint16
	State         string
	UID           uint32
	User          string
	Inode         uint64
	IfIndex       uint32
	Interface     string
	PID           int
	PPID          int
	Process       string
	Executable    string
	Command       string
	Service       string
	NetNS         string
	Container     string
	StartTicks    uint64
	OwnerKnown    bool
	Exposure      string
	Source        string
	BytesSent     uint64
	BytesReceived uint64
	BytesKnown    bool
	SegsOut       uint32
	SegsIn        uint32
	RTTMicrosec   uint32
}

// Key identifies the socket across refreshes.
func (s Socket) Key() string {
	return s.Protocol + "/" + s.Family + "/" + s.LocalIP.String() + "/" + itoa(int(s.LocalPort)) + "/" + uitoa(s.Inode)
}

// IsBoundService reports a TCP listener or any UDP socket.
// Other TCP states are active connections, not published ports.
func (s Socket) IsBoundService() bool {
	if s.Protocol == UDP {
		return true
	}
	return s.Protocol == TCP && s.State == "LISTEN"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func uitoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
