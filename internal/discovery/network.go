package discovery

import (
	"fmt"
	"net"
	"strings"
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

// DefaultPrivateScanCIDR derives the most likely household IPv4 subnet.
// Virtual interfaces are kept as a fallback but physical Wi-Fi/Ethernet names
// win when several private addresses exist.
func DefaultPrivateScanCIDR() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("discovery: list interfaces: %w", err)
	}
	bestCIDR, bestScore := "", -1
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipnet.IP.To4()
			if !isRFC1918(ip4) {
				continue
			}
			ones, bits := ipnet.Mask.Size()
			if bits != 32 {
				continue
			}
			cidr := PrivateScanCIDR(ip4, ones)
			score := interfaceScanScore(iface.Name)
			if score > bestScore {
				bestCIDR, bestScore = cidr, score
			}
		}
	}
	if bestCIDR == "" {
		return "", fmt.Errorf("discovery: no private LAN cidr")
	}
	return bestCIDR, nil
}

// PrivateScanCIDR keeps scans bounded to at most one /24 around the local IP.
func PrivateScanCIDR(ip net.IP, prefix int) string {
	ip4 := ip.To4()
	if ip4 == nil {
		return ""
	}
	if prefix < 24 || prefix > 30 {
		prefix = 24
	}
	mask := net.CIDRMask(prefix, 32)
	return (&net.IPNet{IP: ip4.Mask(mask), Mask: mask}).String()
}

func interfaceScanScore(name string) int {
	n := strings.ToLower(name)
	switch {
	case strings.HasPrefix(n, "en"),
		strings.HasPrefix(n, "eth"),
		strings.HasPrefix(n, "wlan"),
		strings.Contains(n, "wi-fi"),
		strings.Contains(n, "ethernet"):
		return 10
	case strings.HasPrefix(n, "utun"),
		strings.HasPrefix(n, "bridge"),
		strings.HasPrefix(n, "docker"),
		strings.HasPrefix(n, "veth"):
		return 0
	default:
		return 5
	}
}

func isRFC1918(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	return ip4[0] == 10 ||
		(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
		(ip4[0] == 192 && ip4[1] == 168)
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

func interfaceBroadcasts(port int) []*net.UDPAddr {
	var out []*net.UDPAddr
	seen := map[string]struct{}{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 ||
			iface.Flags&net.FlagBroadcast == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok || !isRFC1918(ipnet.IP) {
				continue
			}
			broadcast := ipv4Broadcast(ipnet.IP, ipnet.Mask)
			if broadcast == nil {
				continue
			}
			key := broadcast.String()
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, &net.UDPAddr{IP: broadcast, Port: port})
		}
	}
	return out
}

func ipv4Broadcast(ip net.IP, mask net.IPMask) net.IP {
	ip4 := ip.To4()
	if ip4 == nil || len(mask) != net.IPv4len {
		return nil
	}
	out := make(net.IP, net.IPv4len)
	for i := range out {
		out[i] = ip4[i] | ^mask[i]
	}
	return out
}
