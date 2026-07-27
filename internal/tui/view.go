// Package tui renders the Linux text UI over the engine loopback API.
package tui

import (
	"fmt"
	"strings"
)

// EmptyPeersCopy is shown when online peers list is empty (DUD-UI-120 / P040).
const EmptyPeersCopy = "НИКОГО РЯДОМ"

// PeerRow is one neighbor for the peers pane.
type PeerRow struct {
	PeerID      string
	DisplayName string
}

// Snapshot is one frame of status + peers (P040).
type Snapshot struct {
	MeName     string
	PeerID     string
	Peers      []PeerRow
	ProtoMajor int
	ProtoMinor int
	EngineOK   bool
	Err        string
}

// Render builds a text frame: status strip + peers (or empty copy).
func Render(s Snapshot) string {
	var b strings.Builder
	me := strings.TrimSpace(s.MeName)
	if me == "" {
		me = "—"
	}
	if !s.EngineOK {
		fmt.Fprintf(&b, "ДУДКА · ENGINE OFFLINE\n")
		if s.Err != "" {
			fmt.Fprintf(&b, "  %s\n", s.Err)
		}
		return b.String()
	}
	n := len(s.Peers)
	state := "ok"
	if n == 0 {
		state = "alone"
	}
	fmt.Fprintf(&b, "ДУДКА · %s · online %d · %s", me, n, state)
	if s.ProtoMajor > 0 {
		fmt.Fprintf(&b, " · proto %d.%d", s.ProtoMajor, s.ProtoMinor)
	}
	b.WriteByte('\n')
	b.WriteString("PEERS\n")
	if n == 0 {
		fmt.Fprintf(&b, "  %s\n", EmptyPeersCopy)
		return b.String()
	}
	for _, p := range s.Peers {
		name := strings.TrimSpace(p.DisplayName)
		if name == "" {
			name = p.PeerID
		}
		fmt.Fprintf(&b, "  %s\n", name)
	}
	return b.String()
}
