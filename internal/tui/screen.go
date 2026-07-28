package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// ScreenState is pure view input for the interactive TUI (P046).
type ScreenState struct {
	Snap       Snapshot
	Compose    string
	StatusMsg  string
	FeedScroll int
	CursorOn   bool
}

// RenderScreen paints fixed panels for width×height (no I/O). DESIGN.md charcoal + RU labels.
func RenderScreen(st ScreenState, width, height int) string {
	lay := LayoutFor(width, height)
	rows := make([]string, 0, height)

	rows = append(rows, paintStatus(st.Snap, lay.Width))

	peerLines := peerPaneLines(st.Snap, lay.PeersW, lay.BodyH)
	feedLines := feedPaneLines(st.Snap, lay.FeedW, lay.BodyH, st.FeedScroll)
	for i := 0; i < lay.BodyH; i++ {
		p := peerLines[i]
		f := feedLines[i]
		sep := styleDim().Background(colorPanel).Render("│")
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, p, sep, f))
	}

	rows = append(rows, paintCompose(st.Compose, st.CursorOn, lay.Width))
	rows = append(rows, paintHelp(st.Snap, st.StatusMsg, lay.Width))

	for len(rows) < height {
		rows = append(rows, fillBg("", width, colorPanel))
	}
	if len(rows) > height {
		rows = rows[:height]
	}
	return strings.Join(rows, "\n")
}

func paintStatus(snap Snapshot, width int) string {
	if !snap.EngineOK {
		return fillBg(styleErr().Render(truncateRunes(statusText(snap), width)), width, colorPanel)
	}
	brand := styleBrand().Render(" ◆ ДУДКА ")
	status := styleStatus().Render(truncateRunes(strings.TrimPrefix(statusText(snap), "ДУДКА · "), width-10))
	return fillBg(brand+status, width, colorPanel)
}

func statusText(snap Snapshot) string {
	if !snap.EngineOK {
		text := "ДУДКА · ДВИЖОК НЕДОСТУПЕН"
		if snap.Err != "" {
			text += " · " + snap.Err
		}
		return text
	}
	remoteCount := len(snap.Peers)
	state := DisplayNetworkState(snap.Network, remoteCount)
	me := strings.TrimSpace(snap.MeName)
	if me == "" {
		me = "—"
	}
	// Segment-style brand + online count (DESIGN status strip).
	onlineCount := remoteCount + 1
	text := fmt.Sprintf("ДУДКА · %s · онлайн %d · %s", me, onlineCount, state)
	if snap.ProtoMajor > 0 {
		text += fmt.Sprintf(" · прото %d.%d", snap.ProtoMajor, snap.ProtoMinor)
	}
	return text + "  " + StepPads(onlinePercent(onlineCount))
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

func peerPaneLines(snap Snapshot, width, height int) []string {
	if width < 8 {
		width = 8
	}
	lines := make([]string, 0, height)
	lines = append(lines, fillBg(styleLabel().Render(silkLabel("СОСЕДИ")), width, colorPanelDeep))
	inner := height - 1
	switch {
	case !snap.EngineOK:
		lines = append(lines, fillBg(styleDim().Render("—"), width, colorPanelDeep))
	default:
		me := strings.TrimSpace(snap.MeName)
		if me == "" {
			me = "—"
		}
		selfRow := stylePeerActive().Render("●") +
			styleBody().Render(" "+truncateRunes(me+" · ВЫ", width-2))
		lines = append(lines, fillBg(selfRow, width, colorPanelDeep))
		remaining := inner - 1
		switch {
		case remaining <= 0:
		case snap.Network == NetworkNoNetwork:
			lines = append(lines, fillBg(styleDim().Render(NoNetworkCopy), width, colorPanelDeep))
		case len(snap.Peers) == 0:
			lines = append(lines, fillBg(styleDim().Render(EmptyPeersCopy), width, colorPanelDeep))
			if remaining > 1 {
				lines = append(lines, fillBg(styleAction().Render(" S · ИСКАТЬ "), width, colorPanelDeep))
			}
		default:
			for i, p := range snap.Peers {
				if i >= remaining {
					break
				}
				name := strings.TrimSpace(p.DisplayName)
				if name == "" {
					name = p.PeerID
				}
				row := stylePeerActive().Render("●") + styleBody().Render(" "+truncateRunes(name, width-2))
				lines = append(lines, fillBg(row, width, colorPanelDeep))
			}
		}
	}
	for len(lines) < height {
		lines = append(lines, fillBg("", width, colorPanelDeep))
	}
	return lines[:height]
}

func feedPaneLines(snap Snapshot, width, height, scroll int) []string {
	if width < 12 {
		width = 12
	}
	lines := make([]string, 0, height)
	lines = append(lines, fillBg(styleLabel().Render(silkLabel("ЛЕНТА")), width, colorPanelDeep))
	inner := height - 1
	if inner < 1 {
		inner = 1
	}
	switch {
	case !snap.EngineOK:
		lines = append(lines, fillBg(styleDim().Render("нет данных"), width, colorPanelDeep))
	case len(snap.Messages) == 0:
		lines = append(lines, fillBg(styleDim().Render("— пусто —"), width, colorPanelDeep))
	default:
		progress := map[string]TransferRow{}
		for _, tr := range snap.Transfers {
			progress[tr.FileID] = tr
		}
		raw := make([]string, 0, len(snap.Messages))
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
				ts := styleDim().Render(m.TS.UTC().Format("15:04"))
				sender := styleSender().Render(" " + name)
				rest := styleBody().Render("  " + body)
				line = ts + sender + rest
			}
			raw = append(raw, line)
		}
		start := len(raw) - inner - scroll
		if start < 0 {
			start = 0
		}
		end := start + inner
		if end > len(raw) {
			end = len(raw)
		}
		for _, line := range raw[start:end] {
			lines = append(lines, fillBg(line, width, colorPanelDeep))
		}
	}
	for len(lines) < height {
		lines = append(lines, fillBg("", width, colorPanelDeep))
	}
	return lines[:height]
}

func paintCompose(compose string, cursor bool, width int) string {
	cur := " "
	if cursor {
		cur = styleAction().Render("▌")
	}
	left := styleAction().Render("  ↗ ОТПРАВИТЬ  ")
	right := styleCompose().Render("› " + compose)
	joined := lipgloss.JoinHorizontal(lipgloss.Center, left, right, cur)
	return fillBg(joined, width, colorPanelDeep)
}

func paintHelp(snap Snapshot, statusMsg string, width int) string {
	return fillBg(styleDim().Render(truncateRunes(helpText(snap, statusMsg), width)), width, colorPanel)
}

func helpText(snap Snapshot, statusMsg string) string {
	base := "ENTER отправить  ↑↓ лента  /nick Имя  /announce путь  ESC выход"
	if snap.EngineOK && snap.Network != NetworkNoNetwork && len(snap.Peers) == 0 {
		base = "S искать · " + base
	}
	if msg := strings.TrimSpace(statusMsg); msg != "" {
		return msg + " · " + base
	}
	return base
}

func silkLabel(s string) string {
	// Light silkscreen tracking without CRT noise: thin gaps between letters.
	r := []rune(strings.TrimSpace(s))
	if len(r) == 0 {
		return s
	}
	var b strings.Builder
	for i, ch := range r {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(ch)
	}
	return b.String()
}

func fillBg(content string, width int, bg lipgloss.Color) string {
	if width < 1 {
		width = 1
	}
	plain := stripANSI(content)
	w := runewidth.StringWidth(plain)
	pad := width - w
	if pad < 0 {
		// Truncate styled content roughly by plain width.
		content = styleBody().Background(bg).Foreground(colorSilk).Render(truncateRunes(plain, width))
		plain = stripANSI(content)
		w = runewidth.StringWidth(plain)
		pad = width - w
	}
	if pad < 0 {
		pad = 0
	}
	padStr := lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", pad))
	base := lipgloss.NewStyle().Background(bg).Render("")
	_ = base
	if content == "" {
		return lipgloss.NewStyle().Width(width).Background(bg).Render(strings.Repeat(" ", width))
	}
	return content + padStr
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == 0x1b {
			inEsc = true
			continue
		}
		if inEsc {
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
				inEsc = false
			}
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func truncateRunes(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= width {
		return s
	}
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if w+rw >= width {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	if width > 1 && utf8.RuneCountInString(b.String()) > 0 {
		return b.String() + "…"
	}
	return b.String()
}
