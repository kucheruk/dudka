package discovery

import (
	"net"
	"testing"
)

func TestIPv4Broadcast(t *testing.T) {
	t.Parallel()
	if got := ipv4Broadcast(net.ParseIP("192.168.212.59"), net.CIDRMask(24, 32)); !got.Equal(net.ParseIP("192.168.212.255")) {
		t.Fatalf("broadcast=%v want 192.168.212.255", got)
	}
	if got := ipv4Broadcast(net.ParseIP("10.4.5.6"), net.CIDRMask(16, 32)); !got.Equal(net.ParseIP("10.4.255.255")) {
		t.Fatalf("broadcast=%v want 10.4.255.255", got)
	}
}
