package discovery_test

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"dudka/internal/discovery"
)

func TestCompatibleProtoMajor(t *testing.T) {
	t.Parallel()
	if !discovery.CompatibleProto(1, 1) {
		t.Fatal("same major should match")
	}
	if discovery.CompatibleProto(1, 2) {
		t.Fatal("different major should not match")
	}
}

func TestRegisterRejectsIncompatibleProto(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var logs []string
	store := discovery.NewPeerStore()
	_ = store.Upsert(discovery.Peer{
		PeerID: "good", DisplayName: "Good", InstanceID: "g1",
		Host: "127.0.0.1", TCPPort: 1, ProtoMajor: 1,
	})

	node := discovery.NewNode(discovery.Config{
		PeerID:      "local",
		DisplayName: "Local",
		InstanceID:  "l1",
		ProtoMajor:  1,
		Bind:        "127.0.0.1:0",
		TCPBind:     "127.0.0.1:0",
		Interval:    time.Hour,
		Peers:       store,
		Logf: func(format string, args ...any) {
			mu.Lock()
			logs = append(logs, fmt.Sprintf(format, args...))
			mu.Unlock()
		},
	})
	if err := node.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = node.Stop() })

	addr := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", node.TCPPort()))
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))

	raw, err := discovery.EncodeRegister(discovery.Register{
		Type:        "register",
		PeerID:      "alien",
		DisplayName: "Alien",
		ProtoMajor:  99,
		ProtoMinor:  0,
		TCPPort:     9,
		InstanceID:  "a1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Write(raw); err != nil {
		t.Fatal(err)
	}
	br := bufio.NewReader(conn)
	line, err := br.ReadBytes('\n')
	if err != nil {
		t.Fatal(err)
	}
	resp, err := discovery.DecodeRegister(line)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Type != "register_reject" {
		t.Fatalf("type=%q want register_reject", resp.Type)
	}

	list := store.List()
	if len(list) != 1 || list[0].PeerID != "good" {
		t.Fatalf("peers corrupted: %+v", list)
	}

	mu.Lock()
	joined := strings.Join(logs, "\n")
	mu.Unlock()
	if !strings.Contains(joined, "proto_mismatch") || !strings.Contains(joined, "peer_id=alien") {
		t.Fatalf("missing proto_mismatch log: %q", joined)
	}
	st := node.Status()
	if len(st.Incompatible) != 1 || st.Incompatible[0].PeerID != "alien" {
		t.Fatalf("status=%+v", st)
	}
}
