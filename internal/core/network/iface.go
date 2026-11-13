// Package network describes host interfaces observed at scan time.
package network

import "net/netip"

// Interface is one link and the addresses read from the host.
type Interface struct {
	Index     int
	Name      string
	MTU       int
	Up        bool
	Addresses []netip.Addr
}
