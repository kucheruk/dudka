package tui_test

import (
	"strings"
	"testing"

	"dudka/internal/tui"
)

func TestRenderEmptyPeersShowsNikogoRyadom(t *testing.T) {
	t.Parallel()
	out := tui.Render(tui.Snapshot{
		MeName:   "Вася",
		PeerID:   "peer-1",
		EngineOK: true,
		Peers:    nil,
	})
	if !strings.Contains(out, tui.EmptyPeersCopy) {
		t.Fatalf("want %q in:\n%s", tui.EmptyPeersCopy, out)
	}
	if !strings.Contains(out, "ДУДКА") {
		t.Fatalf("missing brand:\n%s", out)
	}
	if !strings.Contains(out, "Вася") {
		t.Fatalf("missing me name:\n%s", out)
	}
	if !strings.Contains(out, "online 0") {
		t.Fatalf("missing online count:\n%s", out)
	}
}

func TestRenderListsPeers(t *testing.T) {
	t.Parallel()
	out := tui.Render(tui.Snapshot{
		MeName:   "Вася",
		EngineOK: true,
		Peers: []tui.PeerRow{
			{PeerID: "b", DisplayName: "Боб"},
			{PeerID: "c", DisplayName: "Катя"},
		},
	})
	if strings.Contains(out, tui.EmptyPeersCopy) {
		t.Fatalf("must not show empty copy when peers present:\n%s", out)
	}
	if !strings.Contains(out, "online 2") {
		t.Fatalf("missing online 2:\n%s", out)
	}
	if !strings.Contains(out, "Боб") || !strings.Contains(out, "Катя") {
		t.Fatalf("missing peer names:\n%s", out)
	}
}

func TestRenderEngineDown(t *testing.T) {
	t.Parallel()
	out := tui.Render(tui.Snapshot{EngineOK: false, Err: "connection refused"})
	if !strings.Contains(out, "engine") && !strings.Contains(out, "ENGINE") {
		t.Fatalf("want engine error hint:\n%s", out)
	}
}
