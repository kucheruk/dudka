// Package tui renders the Linux text UI over the engine loopback API.
package tui

import (
	"fmt"
	"strings"
	"time"
)

// EmptyPeersCopy is shown when LAN is up but online peers list is empty (DUD-UI-120 / P040).
const EmptyPeersCopy = "НИКОГО РЯДОМ"

// NoNetworkCopy is shown when there is no usable Wi‑Fi/LAN (DUD-UI-120 / P044).
const NoNetworkCopy = "НЕТ СЕТИ"

// AloneHint is the alone-state affordance for subnet scan (DUD-UI-120).
const AloneHint = "ИСКАТЬ"

// Network state mirrors engine GET /status (DUD-NET-140).
const (
	NetworkOK        = "ok"
	NetworkNoNetwork = "no_network"
)

// PeerRow is one neighbor for the peers pane.
type PeerRow struct {
	PeerID      string
	DisplayName string
}

// Feed row types (mirror engine message.type).
const (
	MsgTypeChat         = "chat"
	MsgTypeFileAnnounce = "file_announce"
)

// Transfer statuses mirrored from engine (P052).
const (
	TransferDownloading = "downloading"
	TransferDone        = "done"
	TransferError       = "error"
	TransferCancelled   = "cancelled"
)

// MsgRow is one feed line for the feed pane (text P041, file announce P050).
type MsgRow struct {
	DisplayName string
	Text        string
	TS          time.Time
	Type        string
	FileID      string
	FileName    string
	Size        int64
	Mime        string
	Hash        string
	ThumbPath   string // local preview path when present (P056)
}

// TransferRow is download progress for a file_id (P052).
type TransferRow struct {
	FileID  string
	Name    string
	Percent int
	Status  string
}

// Snapshot is one frame of status + peers + feed.
type Snapshot struct {
	MeName     string
	PeerID     string
	Peers      []PeerRow
	Messages   []MsgRow
	Transfers  []TransferRow
	Network    string // ok | no_network; empty treated as ok
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
	switch {
	case s.Network == NetworkNoNetwork:
		state = "no_network"
	case n == 0:
		state = "alone"
	}
	fmt.Fprintf(&b, "ДУДКА · %s · online %d · %s", me, n, state)
	if s.ProtoMajor > 0 {
		fmt.Fprintf(&b, " · proto %d.%d", s.ProtoMajor, s.ProtoMinor)
	}
	b.WriteByte('\n')
	b.WriteString("PEERS\n")
	switch state {
	case "no_network":
		fmt.Fprintf(&b, "  %s\n", NoNetworkCopy)
	case "alone":
		fmt.Fprintf(&b, "  %s\n", EmptyPeersCopy)
		fmt.Fprintf(&b, "  (%s — subnet scan)\n", AloneHint)
	default:
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
		progress := map[string]TransferRow{}
		for _, tr := range s.Transfers {
			progress[tr.FileID] = tr
		}
		for _, m := range s.Messages {
			name := strings.TrimSpace(m.DisplayName)
			if name == "" {
				name = "—"
			}
			line := feedLine(m, progress[m.FileID])
			ts := m.TS
			if ts.IsZero() {
				fmt.Fprintf(&b, "  · %s · %s\n", name, line)
				continue
			}
			fmt.Fprintf(&b, "  %s · %s · %s\n", ts.UTC().Format("15:04"), name, line)
		}
	}
	b.WriteString("INPUT\n")
	b.WriteString("  >  (Enter = send · /nick Имя · /fetch <file_id> · /fetch! <file_id> · /cancel <file_id>)\n")
	return b.String()
}

func feedLine(m MsgRow, tr TransferRow) string {
	if m.Type == MsgTypeFileAnnounce || (m.FileID != "" && m.FileName != "") {
		name := strings.TrimSpace(m.FileName)
		if name == "" {
			name = "file"
		}
		line := fmt.Sprintf("FILE %s %d %s %s", name, m.Size, strings.TrimSpace(m.Mime), m.FileID)
		if p := strings.TrimSpace(m.ThumbPath); p != "" {
			line = fmt.Sprintf("%s THUMB %s", line, p)
		}
		switch tr.Status {
		case TransferCancelled:
			line = fmt.Sprintf("%s CANCELLED discarded", line)
		case TransferError:
			line = fmt.Sprintf("%s ERROR", line)
		case TransferDownloading, TransferDone:
			if tr.FileID != "" {
				line = fmt.Sprintf("%s %d%%", line, tr.Percent)
			}
		default:
			if tr.FileID != "" {
				line = fmt.Sprintf("%s %d%%", line, tr.Percent)
			} else if IsLargeFile(m.Size) {
				line = fmt.Sprintf("%s WARN>100MiB", line)
			}
		}
		return line
	}
	return strings.TrimSpace(m.Text)
}
