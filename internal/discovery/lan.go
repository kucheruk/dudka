package discovery

import (
	"fmt"
	"net"
	"strings"
)

// AllowedDialIP reports whether engine may open an outbound connection to ip (DUD-NET-101).
// Permitted: loopback, link-local, RFC1918, Unique Local (fc00::/7).
func AllowedDialIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 10 {
			return true
		}
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		return false
	}
	// Unique Local Address fc00::/7
	return len(ip) >= 1 && (ip[0]&0xfe) == 0xfc
}

// CheckDialHost rejects hostnames and non-LAN IP literals before any dial (DUD-NET-101).
func CheckDialHost(host string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("discovery: empty dial host")
	}
	// Unwrap host:port if a caller passed a full address.
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("discovery: refuse non-ip host %q", host)
	}
	if !AllowedDialIP(ip) {
		return fmt.Errorf("discovery: refuse wan address %s", ip)
	}
	return nil
}

// FormatWanRefuse is the structured log line when a WAN dial is blocked.
func FormatWanRefuse(host string) string {
	return fmt.Sprintf("wan_refuse host=%s", host)
}
