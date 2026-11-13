package ports

import (
	"sort"
	"strconv"
)

// Peer is one remote endpoint a local socket is exchanging data with.
type Peer struct {
	State         string
	Remote        string
	LocalPort     uint16
	LocalIP       string
	BytesSent     uint64
	BytesReceived uint64
	BytesKnown    bool
	OnHost        bool
	SegsOut       uint32
	SegsIn        uint32
	RTTMicrosec   uint32
}

// Hub is one local process and port, with every peer from the current scan.
type Hub struct {
	Key       string
	Title     string
	Detail    string
	Peers     []Peer
	Sent      uint64
	Recv      uint64
	Known     bool
	PID       int
	Process   string
	Protocol  string
	LocalIP   string
	LocalPort uint16
	Sockets   []Socket
}

// Topology groups sockets that have a remote peer.
// A listening socket without a peer is not a flow, and neither is an unconnected UDP bind.
func Topology(socks []Socket) []Hub {
	groups := map[string]*Hub{}
	var order []string
	for _, s := range socks {
		if !HasPeer(s) {
			continue
		}
		key := HubKey(s)
		h, ok := groups[key]
		if !ok {
			h = &Hub{
				Key:       key,
				Title:     hubTitle(s),
				Detail:    hubDetail(s),
				PID:       s.PID,
				Process:   s.Process,
				Protocol:  s.Protocol,
				LocalIP:   s.LocalIP.String(),
				LocalPort: s.LocalPort,
			}
			groups[key] = h
			order = append(order, key)
		} else {
			if h.PID == 0 && s.PID > 0 {
				h.PID = s.PID
				h.Detail = hubDetail(s)
			}
			if (h.Process == "" || h.Process == "owner unknown") && s.Process != "" && s.Process != "owner unknown" {
				h.Process = s.Process
				h.Title = hubTitle(s)
			}
			if h.LocalPort != 0 && h.LocalPort != s.LocalPort {
				h.LocalPort = 0
				if h.PID == 0 && !isClosingState(s.State) {
					h.Detail = s.Protocol + " · outbound"
				}
			}
		}
		h.Sockets = append(h.Sockets, s)
		h.Peers = append(h.Peers, Peer{
			State:         s.State,
			Remote:        s.RemoteIP.String() + ":" + strconv.Itoa(int(s.RemotePort)),
			LocalPort:     s.LocalPort,
			LocalIP:       s.LocalIP.String(),
			BytesSent:     s.BytesSent,
			BytesReceived: s.BytesReceived,
			BytesKnown:    s.BytesKnown,
			OnHost:        s.RemoteIP.IsLoopback(),
			SegsOut:       s.SegsOut,
			SegsIn:        s.SegsIn,
			RTTMicrosec:   s.RTTMicrosec,
		})
		if s.BytesKnown {
			h.Sent += s.BytesSent
			h.Recv += s.BytesReceived
			h.Known = true
		}
	}
	out := make([]Hub, 0, len(order))
	for _, key := range order {
		h := groups[key]
		sort.SliceStable(h.Peers, func(i, j int) bool {
			if h.Peers[i].BytesSent != h.Peers[j].BytesSent {
				return h.Peers[i].BytesSent > h.Peers[j].BytesSent
			}
			return h.Peers[i].Remote < h.Peers[j].Remote
		})
		out = append(out, *h)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Sent != out[j].Sent {
			return out[i].Sent > out[j].Sent
		}
		if out[i].Title != out[j].Title {
			return out[i].Title < out[j].Title
		}
		return out[i].Detail < out[j].Detail
	})
	return out
}

// HasPeer reports whether a socket represents an active connected flow with a destination.
func HasPeer(s Socket) bool {
	if s.Protocol == TCP && s.State == "LISTEN" {
		return false
	}
	if s.Protocol == UDP && s.State != "ESTABLISHED" {
		return false
	}
	if !s.RemoteIP.IsValid() || s.RemoteIP.IsUnspecified() || s.RemotePort == 0 {
		return false
	}
	return true
}

func isClosingState(state string) bool {
	switch state {
	case "FIN_WAIT_1", "FIN_WAIT_2", "FIN-WAIT-1", "FIN-WAIT-2",
		"TIME_WAIT", "TIME-WAIT", "LAST_ACK", "LAST-ACK",
		"CLOSE_WAIT", "CLOSE-WAIT", "CLOSING":
		return true
	default:
		return false
	}
}

// HubKey computes the topology grouping key for a socket.
func HubKey(s Socket) string {
	if s.PID > 0 {
		return s.Protocol + "/pid/" + strconv.Itoa(s.PID)
	}
	if s.Process != "" && s.Process != "owner unknown" {
		return s.Protocol + "/proc/" + s.Process
	}
	if s.Service != "" {
		return s.Protocol + "/svc/" + s.Service
	}
	if isClosingState(s.State) {
		return s.Protocol + "/kernel/closing"
	}
	if s.User != "" && s.User != "unknown" {
		if s.LocalPort > 0 && (s.State == "LISTEN" || s.State == "UNCONN") {
			return s.Protocol + "/unattributed/" + s.User + "/:" + strconv.Itoa(int(s.LocalPort))
		}
		return s.Protocol + "/unattributed/" + s.User
	}
	if s.LocalPort > 0 && (s.State == "LISTEN" || s.State == "UNCONN") {
		return s.Protocol + "/unattributed/:" + strconv.Itoa(int(s.LocalPort))
	}
	return s.Protocol + "/unattributed"
}

func hubTitle(s Socket) string {
	if s.Process != "" && s.Process != "owner unknown" {
		return s.Process
	}
	if s.Service != "" {
		return s.Service
	}
	if isClosingState(s.State) {
		return "kernel (closing sockets)"
	}
	if s.User != "" && s.User != "unknown" {
		if s.User == "root" {
			return "system (root)"
		}
		return s.User + " (outbound)"
	}
	return "owner unknown"
}

func hubDetail(s Socket) string {
	if s.PID > 0 {
		return "PID " + strconv.Itoa(s.PID) + " · " + s.Protocol
	}
	if isClosingState(s.State) {
		return "TCP Teardown · " + s.Protocol
	}
	if s.LocalPort > 0 && (s.State == "LISTEN" || s.State == "UNCONN") {
		return s.Protocol + " · :" + strconv.Itoa(int(s.LocalPort))
	}
	return s.Protocol + " · outbound"
}
