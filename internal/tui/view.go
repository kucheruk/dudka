// Package tui renders the Linux text UI over the engine loopback API.
package tui

import (
	"fmt"
	"strings"
	"time"
)

// EmptyPeersCopy is shown when online peers list is empty (DUD-UI-120 / P040).
const EmptyPeersCopy = "НИКОГО РЯДОМ"

// PeerRow is one neighbor for the peers pane.
type PeerRow struct {
	PeerID      string
	DisplayName string
}

// MsgRow is one chat line for the feed pane (P041).
type MsgRow struct {
	DisplayName string
	Text        string
	TS          time.Time
}

// Snapshot is one frame of status + peers + feed.
type Snapshot struct {
	MeName     string
	PeerID     string
	Peers      []PeerRow
	Messages   []MsgRow
	ProtoMajor int
	ProtoMinor int
	EngineOK   bool
	Err        string
}

// Render builds a text frame: status strip + peers + message feed.
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
	} else {
		for _, p := range s.Peers {
			name := strings.TrimSpace(p.DisplayName)
			if name == "" {
				name = p.PeerID
			}
			fmt.Fprintf(&b, "  %s\n", name)
		}
	}
	b.WriteString("FEED\n")
	if len(s.Messages) == 0 {
		b.WriteString("  —\n")
	} else {
		for _, m := range s.Messages {
			name := strings.TrimSpace(m.DisplayName)
			if name == "" {
				name = "—"
			}
			text := strings.TrimSpace(m.Text)
			ts := m.TS
			if ts.IsZero() {
				fmt.Fprintf(&b, "  · %s · %s\n", name, text)
				continue
			}
			fmt.Fprintf(&b, "  %s · %s · %s\n", ts.UTC().Format("15:04"), name, text)
		}
	}
	b.WriteString("INPUT\n")
	b.WriteString("  >  (Enter = send)\n")
	return b.String()
}
