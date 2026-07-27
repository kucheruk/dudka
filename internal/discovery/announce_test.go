package discovery_test

import (
	"strings"
	"testing"

	"dudka/internal/discovery"
)

func TestAnnounceRoundTrip(t *testing.T) {
	t.Parallel()
	in := discovery.Announce{
		PeerID:      "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
		DisplayName: "Вася",
		ProtoMajor:  1,
		ProtoMinor:  0,
		TCPPort:     41777,
		InstanceID:  "ffffffff-1111-4222-8333-444444444444",
	}
	raw, err := discovery.EncodeAnnounce(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := discovery.DecodeAnnounce(raw)
	if err != nil {
		t.Fatal(err)
	}
	in.Type = "announce"
	if out != in {
		t.Fatalf("round-trip: got %+v want %+v", out, in)
	}
}

func TestDecodeAnnounceRejectsGarbage(t *testing.T) {
	t.Parallel()
	if _, err := discovery.DecodeAnnounce([]byte("not-json")); err == nil {
		t.Fatal("expected error")
	}
}

func TestFormatAnnounceRx(t *testing.T) {
	t.Parallel()
	line := discovery.FormatAnnounceRx(discovery.Announce{
		PeerID:      "p1",
		DisplayName: "Ник",
	}, "192.168.1.5:41777")
	if !strings.Contains(line, "announce_rx") || !strings.Contains(line, "peer_id=p1") {
		t.Fatalf("line=%q", line)
	}
}
