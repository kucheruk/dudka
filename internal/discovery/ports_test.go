package discovery

import (
	"net"
	"testing"
	"time"
)

func TestListenTCPWithFallbackWhenBusy(t *testing.T) {
	ln1, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln1.Close()
	port := ln1.Addr().(*net.TCPAddr).Port

	ln2, got, relocated, err := listenTCPWithFallback(port, 5)
	if err != nil {
		t.Fatalf("fallback: %v", err)
	}
	defer ln2.Close()
	if got == port {
		t.Fatalf("expected different port, got same %d", got)
	}
	if !relocated {
		t.Fatal("expected relocated=true")
	}
}

func TestNodeStartTCPPortBusyStillAlive(t *testing.T) {
	blocker, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Close()
	busy := blocker.Addr().(*net.TCPAddr).Port

	n := NewNode(Config{
		PeerID:      "p1",
		DisplayName: "A",
		InstanceID:  "i1",
		TCPPort:     busy,
		Bind:        "127.0.0.1:0",
		Interval:    time.Second,
	})
	if err := n.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = n.Stop() }()
	if n.TCPPort() == busy {
		t.Fatalf("tcp should relocate off busy %d", busy)
	}
	st := n.Status()
	if !st.PortRelocated {
		t.Fatalf("status PortRelocated=false note=%q", st.PortNote)
	}
	if st.SessionPort != n.TCPPort() {
		t.Fatalf("session_port %d != tcp %d", st.SessionPort, n.TCPPort())
	}
	if st.PortNote == "" {
		t.Fatal("expected port_note")
	}
}
