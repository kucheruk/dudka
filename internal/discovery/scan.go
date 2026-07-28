package discovery

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// ScanRequest configures a subnet/host probe used when broadcast is blocked (P024).
type ScanRequest struct {
	Hosts []string `json:"hosts,omitempty"`
	Port  int      `json:"port,omitempty"` // TCP register port to probe
	CIDR  string   `json:"cidr,omitempty"` // optional private CIDR; ignored if Hosts set
}

// ScanResult summarizes what Scan discovered.
type ScanResult struct {
	Probed int    `json:"probed"`
	Found  int    `json:"found"`
	Peers  []Peer `json:"peers"`
}

// Scan probes hosts over TCP register without relying on UDP broadcast.
func (n *Node) Scan(ctx context.Context, req ScanRequest) (ScanResult, error) {
	port := req.Port
	if port <= 0 {
		n.mu.Lock()
		port = n.tcpPort
		if port == 0 {
			port = n.cfg.TCPPort
		}
		n.mu.Unlock()
		if port <= 0 {
			port = DefaultUDPPort
		}
	}

	hosts := append([]string{}, req.Hosts...)
	if len(hosts) == 0 && strings.TrimSpace(req.CIDR) == "" {
		probe := n.cfg.ScanCIDR
		if probe == nil {
			probe = DefaultPrivateScanCIDR
		}
		cidr, err := probe()
		if err != nil {
			return ScanResult{}, err
		}
		req.CIDR = cidr
	}
	if len(hosts) == 0 && req.CIDR != "" {
		expanded, err := ExpandPrivateCIDR(req.CIDR, 256)
		if err != nil {
			return ScanResult{}, err
		}
		hosts = expanded
	}
	if len(hosts) == 0 {
		return ScanResult{}, fmt.Errorf("discovery: scan requires hosts or private cidr")
	}

	selfTCP := n.TCPPort()
	localIPs := localInterfaceIPs()
	var out ScanResult
	seen := map[string]struct{}{}
	candidates := make([]string, 0, len(hosts))
	for _, host := range hosts {
		host = strings.TrimSpace(host)
		if host == "" || host == "0.0.0.0" {
			continue
		}
		// Skip probing our own listener on loopback or a LAN address.
		if _, local := localIPs[host]; local && port == selfTCP {
			continue
		}
		if _, duplicate := seen[host]; duplicate {
			continue
		}
		seen[host] = struct{}{}
		candidates = append(candidates, host)
	}

	type hit struct {
		host string
		peer *Peer
	}
	hits := make(chan hit, len(candidates))
	for _, host := range candidates {
		host := host
		go func() {
			peer, _ := n.dialRegister(host, port)
			select {
			case hits <- hit{host: host, peer: peer}:
			case <-ctx.Done():
			}
		}()
	}

	out.Probed = len(candidates)
	seen = map[string]struct{}{}
	for range candidates {
		select {
		case <-ctx.Done():
			out.Peers = n.cfg.Peers.List()
			return out, ctx.Err()
		case found := <-hits:
			if found.peer == nil {
				continue
			}
			if _, ok := seen[found.peer.PeerID]; ok {
				continue
			}
			seen[found.peer.PeerID] = struct{}{}
			out.Found++
			out.Peers = append(out.Peers, *found.peer)
			n.cfg.Logf(
				"scan_hit peer_id=%s name=%s addr=%s",
				found.peer.PeerID,
				found.peer.DisplayName,
				net.JoinHostPort(found.host, strconv.Itoa(port)),
			)
		}
	}
	if out.Peers == nil {
		out.Peers = []Peer{}
	}
	return out, nil
}

func localInterfaceIPs() map[string]struct{} {
	out := map[string]struct{}{
		"127.0.0.1": {},
		"::1":       {},
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ip := addrIP(addr); ip != nil {
				out[ip.String()] = struct{}{}
			}
		}
	}
	return out
}

// ExpandPrivateCIDR lists host IPs in a private CIDR (capped), for scan fallback.
func ExpandPrivateCIDR(cidr string, limit int) ([]string, error) {
	_, ipnet, err := net.ParseCIDR(strings.TrimSpace(cidr))
	if err != nil {
		return nil, fmt.Errorf("discovery: bad cidr: %w", err)
	}
	if !isPrivateIPNet(ipnet) {
		return nil, fmt.Errorf("discovery: refuse non-private cidr %s", cidr)
	}
	if limit <= 0 {
		limit = 256
	}
	ip := make(net.IP, len(ipnet.IP))
	copy(ip, ipnet.IP.Mask(ipnet.Mask))
	networkIP := append(net.IP(nil), ip...)
	broadcastIP := append(net.IP(nil), networkIP...)
	for i := range broadcastIP {
		broadcastIP[i] |= ^ipnet.Mask[i]
	}
	var hosts []string
	for ; ipnet.Contains(ip); incIP(ip) {
		if ip.To4() != nil && (ip.Equal(networkIP) || ip.Equal(broadcastIP)) {
			continue
		}
		hosts = append(hosts, ip.String())
		if len(hosts) >= limit {
			break
		}
	}
	return hosts, nil
}

func isPrivateIPNet(n *net.IPNet) bool {
	ones, bits := n.Mask.Size()
	if bits != 32 && bits != 128 {
		return false
	}
	if !AllowedDialIP(n.IP) {
		return false
	}
	if n.IP.IsLoopback() || n.IP.IsLinkLocalUnicast() {
		return true
	}
	if ip4 := n.IP.To4(); ip4 != nil {
		if ip4[0] == 10 {
			return ones >= 8
		}
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return ones >= 12
		}
		if ip4[0] == 192 && ip4[1] == 168 {
			return ones >= 16
		}
		return false
	}
	// Unique Local fc00::/7 — require at least /8 worth of prefix discipline.
	return ones >= 8
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// DefaultScanTimeout bounds POST /scan work.
const DefaultScanTimeout = 5 * time.Second
