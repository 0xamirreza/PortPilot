package ports

import "github.com/0xamirreza/portpilot/internal/core/network"

// Summary is the overview count. Counts use bound services only.
type Summary struct {
	TCPListen    int
	UDPBound     int
	Processes    int
	Wildcard4    int
	Wildcard6    int
	Localhost    int
	OwnerUnknown int
	InterfacesUp int
}

// Summarize counts what the overview stamps show.
func Summarize(sockets []Socket, ifaces []network.Interface) Summary {
	var s Summary
	seen := map[int]struct{}{}
	for _, sock := range sockets {
		if !sock.IsBoundService() {
			continue
		}
		if sock.Protocol == TCP {
			s.TCPListen++
		} else if sock.Protocol == UDP {
			s.UDPBound++
		}
		switch sock.Exposure {
		case ExposureWildcard:
			if sock.Family == IPv4 {
				s.Wildcard4++
			} else if sock.Family == IPv6 {
				s.Wildcard6++
			}
		case ExposureLocalhost:
			s.Localhost++
		}
		if !sock.OwnerKnown {
			s.OwnerUnknown++
		}
		if sock.PID > 0 {
			seen[sock.PID] = struct{}{}
		}
	}
	s.Processes = len(seen)
	for _, ifc := range ifaces {
		if ifc.Up {
			s.InterfacesUp++
		}
	}
	return s
}
