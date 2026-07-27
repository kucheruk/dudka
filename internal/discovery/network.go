package discovery

import (
	"net"
)

// Network state values exposed on GET /status (DUD-NET-140 / P044).
const (
	NetworkOK        = "ok"
	NetworkNoNetwork = "no_network"
)

// LANProbe detects a usable non-loopback LAN interface. Tests may replace it.
var LANProbe = DefaultLANProbe

// HasUsableLAN reports whether a live Wi‑Fi/LAN interface is available.
func HasUsableLAN() bool {
	return LANProbe()
}

// DefaultLANProbe returns true when any UP non-loopback iface has a unicast address.
func DefaultLANProbe() bool {
	ifaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}
		for _, a := range addrs {
			ip := addrIP(a)
			if ip == nil {
				continue
			}
			if ip.IsLoopback() {
				continue
			}
			// Link-local or routable unicast counts as "network present".
			if ip.IsLinkLocalUnicast() || ip.IsGlobalUnicast() {
				return true
			}
		}
	}
	return false
}

func addrIP(a net.Addr) net.IP {
	switch v := a.(type) {
	case *net.IPNet:
		return v.IP
	case *net.IPAddr:
		return v.IP
	default:
		return nil
	}
}
