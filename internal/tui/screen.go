package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ScreenState is pure view input for the interactive TUI (P046).
type ScreenState struct {
	Snap       Snapshot
	Compose    string
	StatusMsg  string // transient error / hint under compose
	FeedScroll int    // lines scrolled up from bottom (0 = newest at bottom)
	CursorOn   bool
}

// RenderScreen paints fixed panels for width×height (no I/O). DESIGN.md charcoal + RU labels.
func RenderScreen(st ScreenState, width, height int) string {
	lay := LayoutFor(width, height)
	status := clampWidth(styleStatus().Render(truncateRunes(statusText(st.Snap), lay.Width)), lay.Width)
	peers := renderPeersPane(st.Snap, lay.PeersW, lay.BodyH)
	feed := renderFeedPane(st.Snap, lay.FeedW, lay.BodyH, st.FeedScroll)
	sep := styleDim().Render("│")
	body := lipgloss.JoinHorizontal(lipgloss.Top, peers, sep, feed)
	compose := renderComposeLine(st.Compose, st.CursorOn, lay.Width)
	help := clampWidth(styleDim().Render(truncateRunes(helpText(st.Snap, st.StatusMsg), lay.Width)), lay.Width)

	out := lipgloss.JoinVertical(lipgloss.Left, status, body, compose, help)
	lines := strings.Split(out, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func statusText(snap Snapshot) string {
	if !snap.EngineOK {
		text := "ДУДКА · ДВИЖОК НЕДОСТУПЕН"
		if snap.Err != "" {
			text += " · " + snap.Err
		}
		return text
	}
	n := len(snap.Peers)
	state := DisplayNetworkState(snap.Network, n)
	me := strings.TrimSpace(snap.MeName)
	if me == "" {
		me = "—"
	}
	text := fmt.Sprintf("ДУДКА · %s · онлайн %d · %s", me, n, state)
	if snap.ProtoMajor > 0 {
		text += fmt.Sprintf(" · прото %d.%d", snap.ProtoMajor, snap.ProtoMinor)
	}
	return text + "  " + StepPads(onlinePercent(n))
}

func onlinePercent(n int) int {
	if n <= 0 {
		return 0
	}
	if n >= 4 {
		return 100
	}
	return n * 25
}

func renderPeersPane(snap Snapshot, width, height int) string {
	if width < 8 {
		width = 8
	}
	rows := make([]string, 0, height)
	rows = append(rows, clampWidth(styleLabel().Render(truncateRunes("СОСЕДИ", width)), width))
	inner := height - 1
	switch {
	case !snap.EngineOK:
		rows = append(rows, fillRow(styleDim().Render("—"), width))
	case snap.Network == NetworkNoNetwork:
		rows = append(rows, fillRow(styleDim().Render(NoNetworkCopy), width))
	case len(snap.Peers) == 0:
		rows = append(rows, fillRow(styleDim().Render(EmptyPeersCopy), width))
		if inner > 1 {
			rows = append(rows, fillRow(styleAction().Render(" S · ИСКАТЬ "), width))
		}
	default:
		for i, p := range snap.Peers {
			if i >= inner {
				break
			}
			name := strings.TrimSpace(p.DisplayName)
			if name == "" {
				name = p.PeerID
			}
			row := stylePeerActive().Render("●") + styleBody().Render(" "+truncateRunes(name, width-2))
			rows = append(rows, fillRow(row, width))
		}
	}
	for len(rows) < height {
		rows = append(rows, fillRow("", width))
	}
	if len(rows) > height {
		rows = rows[:height]
	}
	return lipgloss.NewStyle().Width(width).Background(colorPanelDeep).Render(
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
}

func renderFeedPane(snap Snapshot, width, height, scroll int) string {
	if width < 12 {
		width = 12
	}
	rows := make([]string, 0, height)
	rows = append(rows, clampWidth(styleLabel().Render(truncateRunes("ЛЕНТА", width)), width))
	inner := height - 1
	if inner < 1 {
		inner = 1
	}
	switch {
	case !snap.EngineOK:
		rows = append(rows, fillRow(styleDim().Render("нет данных"), width))
	case len(snap.Messages) == 0:
		rows = append(rows, fillRow(styleDim().Render("— пусто —"), width))
	default:
		progress := map[string]TransferRow{}
		for _, tr := range snap.Transfers {
			progress[tr.FileID] = tr
		}
		lines := make([]string, 0, len(snap.Messages))
		for _, m := range snap.Messages {
			name := strings.TrimSpace(m.DisplayName)
			if name == "" {
				name = "—"
			}
			body := feedLine(m, progress[m.FileID])
			var line string
			if m.TS.IsZero() {
				line = fmt.Sprintf("· %s · %s", name, body)
			} else {
				line = fmt.Sprintf("%s · %s · %s", m.TS.UTC().Format("15:04"), name, body)
			}
			lines = append(lines, line)
		}
		start := len(lines) - inner - scroll
		if start < 0 {
			start = 0
		}
		end := start + inner
		if end > len(lines) {
			end = len(lines)
		}
		for _, line := range lines[start:end] {
			rows = append(rows, fillRow(styleBody().Render(truncateRunes(line, width)), width))
		}
	}
	for len(rows) < height {
		rows = append(rows, fillRow("", width))
	}
	if len(rows) > height {
		rows = rows[:height]
	}
	return lipgloss.NewStyle().Width(width).Background(colorPanelDeep).Render(
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
}

func renderComposeLine(compose string, cursor bool, width int) string {
	cur := " "
	if cursor {
		cur = "▌"
	}
	left := styleAction().Render(" ДУНУТЬ ")
	right := styleCompose().Render("› " + compose + cur)
	joined := lipgloss.JoinHorizontal(lipgloss.Center, left, right)
	return lipgloss.NewStyle().Width(width).Background(colorPanel).Render(joined)
}

func helpText(snap Snapshot, statusMsg string) string {
	base := "Enter отправить · ↑↓ лента · /nick Имя · /announce путь · Ctrl+C выход"
	if snap.EngineOK && snap.Network != NetworkNoNetwork && len(snap.Peers) == 0 {
		base = "S искать · " + base
	}
	if msg := strings.TrimSpace(statusMsg); msg != "" {
		return msg + " · " + base
	}
	return base
}

func fillRow(content string, width int) string {
	return lipgloss.NewStyle().Width(width).Background(colorPanelDeep).Render(content)
}

func clampWidth(s string, width int) string {
	return lipgloss.NewStyle().Width(width).MaxWidth(width).Background(colorPanel).Render(s)
}

func truncateRunes(s string, width int) string {
	if width <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width <= 1 {
		return string(r[:width])
	}
	return string(r[:width-1]) + "…"
}
