package ports

import "net/netip"

// ClassifyExposure separates loopback, wildcard, and a single address.
// It does not decide whether a packet from another host can arrive.
func ClassifyExposure(ip netip.Addr) string {
	if !ip.IsValid() {
		return ExposureUnknown
	}
	ip = ip.Unmap()
	switch {
	case ip.IsLoopback():
		return ExposureLocalhost
	case ip.IsUnspecified():
		return ExposureWildcard
	default:
		return ExposureSpecific
	}
}

// Reachability is the sentence shown next to a socket.
func Reachability(s Socket) string {
	switch s.Exposure {
	case ExposureLocalhost:
		return "Local only. The socket is bound to a loopback address."
	case ExposureWildcard:
		return "Bound on every address of this host. That does not show whether another machine can reach it."
	case ExposureSpecific:
		return "Bound to one address. Reachability from another machine is not known from the bind address."
	default:
		return "Exposure is unknown."
	}
}
