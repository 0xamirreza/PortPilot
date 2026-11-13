// Package interfaces reads host network interfaces.
package interfaces

import (
	"net"
	"net/netip"

	"github.com/0xamirreza/portpilot/internal/core/network"
)

// List reads interfaces through the Go net package, which queries the kernel.
func List() ([]network.Interface, error) {
	raw, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	out := make([]network.Interface, 0, len(raw))
	for _, ifc := range raw {
		item := network.Interface{
			Index: ifc.Index,
			Name:  ifc.Name,
			MTU:   ifc.MTU,
			Up:    ifc.Flags&net.FlagUp != 0,
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			out = append(out, item)
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok || ipnet.IP == nil {
				continue
			}
			parsed, ok := netip.AddrFromSlice(ipnet.IP)
			if !ok {
				continue
			}
			item.Addresses = append(item.Addresses, parsed.Unmap())
		}
		out = append(out, item)
	}
	return out, nil
}
