package discovery_test

import (
	"testing"
	"time"

	"dudka/internal/discovery"
)

func TestStatusNetworkNoNetworkWhenLANDown(t *testing.T) {
	prev := discovery.LANProbe
	t.Cleanup(func() { discovery.LANProbe = prev })
	discovery.LANProbe = func() bool { return false }

	node := discovery.NewNode(discovery.Config{
		PeerID: "p1", DisplayName: "A", InstanceID: "i1",
		Bind: "127.0.0.1:0", TCPBind: "127.0.0.1:0",
		Interval: time.Hour,
		Peers:    discovery.NewPeerStore(),
	})
	if err := node.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = node.Stop() })

	st := node.Status()
	if st.Network != discovery.NetworkNoNetwork {
		t.Fatalf("network=%q want %q", st.Network, discovery.NetworkNoNetwork)
	}
}

func TestStatusNetworkOKWhenLANUpAlone(t *testing.T) {
	prev := discovery.LANProbe
	t.Cleanup(func() { discovery.LANProbe = prev })
	discovery.LANProbe = func() bool { return true }

	node := discovery.NewNode(discovery.Config{
		PeerID: "p1", DisplayName: "A", InstanceID: "i1",
		Bind: "127.0.0.1:0", TCPBind: "127.0.0.1:0",
		Interval: time.Hour,
		Peers:    discovery.NewPeerStore(),
	})
	if err := node.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = node.Stop() })

	st := node.Status()
	if st.Network != discovery.NetworkOK {
		t.Fatalf("network=%q want %q (LAN up, peers=0 is alone not no_network)", st.Network, discovery.NetworkOK)
	}
}

func TestHasUsableLANDefaultProbe(t *testing.T) {
	// On a normal developer machine there is usually a non-loopback UP iface.
	// We only assert the probe is callable and returns a bool (no panic).
	_ = discovery.HasUsableLAN()
}
