package discovery_test

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"dudka/internal/discovery"
)

// DUD-NET-101 / P025: public IP from config must never reach the dialer (no WAN connect).
func TestConfigPublicIPDoesNotReachDialer(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var dialed []string
	var logs []string

	n := discovery.NewNode(discovery.Config{
		PeerID:      "guard-peer",
		DisplayName: "Guard",
		InstanceID:  "guard-inst",
		Bind:        "127.0.0.1:0",
		TCPBind:     "127.0.0.1:0",
		Interval:    time.Hour,
		DialHosts:   []string{"8.8.8.8", "1.1.1.1"},
		DialTimeout: 50 * time.Millisecond,
		Dialer: func(network, address string, timeout time.Duration) (net.Conn, error) {
			mu.Lock()
			dialed = append(dialed, address)
			mu.Unlock()
			return nil, errors.New("dial should not be called for WAN")
		},
		Logf: func(format string, args ...any) {
			mu.Lock()
			logs = append(logs, fmt.Sprintf(format, args...))
			mu.Unlock()
		},
	})
	if err := n.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = n.Stop() })

	mu.Lock()
	defer mu.Unlock()
	if len(dialed) != 0 {
		t.Fatalf("WAN dial escaped to dialer: %v", dialed)
	}
	for _, line := range logs {
		if strings.Contains(line, "wan_refuse") {
			return
		}
	}
	t.Fatalf("expected wan_refuse log, got logs=%v", logs)
}

func TestConfigPrivateIPReachesDialer(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var dialed []string

	n := discovery.NewNode(discovery.Config{
		PeerID:      "lan-peer",
		DisplayName: "Lan",
		InstanceID:  "lan-inst",
		Bind:        "127.0.0.1:0",
		TCPBind:     "127.0.0.1:0",
		Interval:    time.Hour,
		DialHosts:   []string{"127.0.0.1"},
		DialTimeout: 50 * time.Millisecond,
		Dialer: func(network, address string, timeout time.Duration) (net.Conn, error) {
			mu.Lock()
			dialed = append(dialed, address)
			mu.Unlock()
			return nil, errors.New("fake dialer: stop after record")
		},
	})
	if err := n.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = n.Stop() })

	mu.Lock()
	defer mu.Unlock()
	if len(dialed) == 0 {
		t.Fatal("expected dialer call for private/loopback seed host")
	}
	for _, addr := range dialed {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			t.Fatalf("bad dial addr %q: %v", addr, err)
		}
		if err := discovery.CheckDialHost(host); err != nil {
			t.Fatalf("dialer saw non-LAN host %q: %v", host, err)
		}
	}
}

func TestAllowedDialIP(t *testing.T) {
	t.Parallel()
	cases := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"172.16.5.1", true},
		{"192.168.1.1", true},
		{"169.254.1.1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"::1", true},
		{"fd12::1", true},
		{"2001:4860:4860::8888", false},
	}
	for _, tc := range cases {
		ip := net.ParseIP(tc.ip)
		if got := discovery.AllowedDialIP(ip); got != tc.want {
			t.Fatalf("AllowedDialIP(%s)=%v want %v", tc.ip, got, tc.want)
		}
	}
}
