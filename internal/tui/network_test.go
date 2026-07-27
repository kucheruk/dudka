package tui_test

import (
	"strings"
	"testing"

	"dudka/internal/tui"
)

func TestRenderAloneCopyDistinctFromNoNetwork(t *testing.T) {
	t.Parallel()
	alone := tui.Render(tui.Snapshot{
		MeName:   "Вася",
		EngineOK: true,
		Network:  tui.NetworkOK,
		Peers:    nil,
	})
	noNet := tui.Render(tui.Snapshot{
		MeName:   "Вася",
		EngineOK: true,
		Network:  tui.NetworkNoNetwork,
		Peers:    nil,
	})

	if !strings.Contains(alone, tui.EmptyPeersCopy) {
		t.Fatalf("alone missing %q:\n%s", tui.EmptyPeersCopy, alone)
	}
	if !strings.Contains(alone, "alone") {
		t.Fatalf("alone missing state token:\n%s", alone)
	}
	if !strings.Contains(alone, tui.AloneHint) {
		t.Fatalf("alone missing %q hint:\n%s", tui.AloneHint, alone)
	}
	if strings.Contains(alone, tui.NoNetworkCopy) {
		t.Fatalf("alone must not show no_network copy:\n%s", alone)
	}

	if !strings.Contains(noNet, tui.NoNetworkCopy) {
		t.Fatalf("no_network missing %q:\n%s", tui.NoNetworkCopy, noNet)
	}
	if !strings.Contains(noNet, "no_network") {
		t.Fatalf("no_network missing state token:\n%s", noNet)
	}
	if strings.Contains(noNet, tui.EmptyPeersCopy) {
		t.Fatalf("no_network must not show alone empty copy:\n%s", noNet)
	}
	if strings.Contains(noNet, "alone") {
		t.Fatalf("no_network must not use alone state:\n%s", noNet)
	}
	if strings.Contains(noNet, tui.AloneHint) {
		t.Fatalf("no_network must not advertise ИСКАТЬ:\n%s", noNet)
	}

	if alone == noNet {
		t.Fatalf("alone and no_network frames must differ")
	}
}

func TestRenderNoNetworkOverridesPeersListCopy(t *testing.T) {
	t.Parallel()
	out := tui.Render(tui.Snapshot{
		MeName:   "Вася",
		EngineOK: true,
		Network:  tui.NetworkNoNetwork,
		Peers:    nil,
	})
	if strings.Contains(out, "online 0") && !strings.Contains(out, "no_network") {
		t.Fatalf("want no_network state with online 0:\n%s", out)
	}
}
