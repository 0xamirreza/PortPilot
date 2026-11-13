package ports

import (
	"strconv"
	"strings"
)

// Query selects which sockets a screen shows.
type Query struct {
	Text      string
	TCP       bool
	UDP       bool
	IPv4      bool
	IPv6      bool
	Localhost bool
	Wildcard  bool
	Specific  bool
	Listening bool
	Active    bool
}

// DefaultQuery shows bound services on every family and exposure.
func DefaultQuery() Query {
	return Query{
		TCP: true, UDP: true,
		IPv4: true, IPv6: true,
		Localhost: true, Wildcard: true, Specific: true,
		Listening: true,
	}
}

// Match reports whether s satisfies q.
func (q Query) Match(s Socket) bool {
	switch s.Protocol {
	case TCP:
		if !q.TCP {
			return false
		}
	case UDP:
		if !q.UDP {
			return false
		}
	default:
		return false
	}
	switch s.Family {
	case IPv4:
		if !q.IPv4 {
			return false
		}
	case IPv6:
		if !q.IPv6 {
			return false
		}
	default:
		return false
	}
	switch s.Exposure {
	case ExposureLocalhost:
		if !q.Localhost {
			return false
		}
	case ExposureWildcard:
		if !q.Wildcard {
			return false
		}
	case ExposureSpecific:
		if !q.Specific {
			return false
		}
	default:
		if !q.Specific && !q.Wildcard && !q.Localhost {
			return false
		}
	}
	if s.IsBoundService() {
		if !q.Listening {
			return false
		}
	} else if !q.Active {
		return false
	}
	if q.Text == "" {
		return true
	}
	needle := strings.ToLower(strings.TrimSpace(q.Text))
	hay := strings.ToLower(strings.Join([]string{
		s.Protocol,
		s.Family,
		s.LocalIP.String(),
		strconv.Itoa(int(s.LocalPort)),
		s.RemoteIP.String(),
		strconv.Itoa(int(s.RemotePort)),
		s.State,
		s.Process,
		s.Executable,
		s.User,
		s.Service,
		s.Container,
		s.Interface,
		strconv.Itoa(s.PID),
	}, " "))
	return strings.Contains(hay, needle)
}

// Apply returns the sockets that match, preserving order.
func Apply(in []Socket, q Query) []Socket {
	out := make([]Socket, 0, len(in))
	for _, s := range in {
		if q.Match(s) {
			out = append(out, s)
		}
	}
	return out
}
