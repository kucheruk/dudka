package tui

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestF5CopiesLastTechnicalErrorWithOSC52(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TMUX", "")

	m := NewModel(NewClient("http://127.0.0.1:9"))
	updated, _ := m.Update(statusErrMsg{
		display: "ПОИСК НЕ ЗАВЕРШЁН · повторите /search",
		detail:  "поиск: context deadline exceeded",
	})
	m = updated.(Model)
	if !strings.Contains(m.View(), "F5 · КОПИРОВАТЬ ДИАГНОСТИКУ") {
		t.Fatalf("error view must advertise F5:\n%s", m.View())
	}
	if strings.Contains(m.View(), "context deadline exceeded") {
		t.Fatalf("technical detail leaked into visible UI:\n%s", m.View())
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyF5})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("F5 must schedule clipboard pulse cleanup")
	}
	payload := osc52Payload(t, m.View())
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("decode OSC52 payload: %v", err)
	}
	report := string(decoded)
	for _, want := range []string{
		"ДУДКА — ДИАГНОСТИКА ДЛЯ АГЕНТА",
		"версия:",
		"платформа:",
		"движок: loopback",
		"ОШИБКА\nпоиск: context deadline exceeded",
		"ХВОСТ tui.log",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("diagnostic report misses %q:\n%s", want, report)
		}
	}
	if strings.Contains(report, "display_name") || strings.Contains(report, "peer_id") {
		t.Fatalf("diagnostic report must not contain chat identity data:\n%s", report)
	}
	if payload == "" {
		t.Fatalf("view does not contain OSC52 clipboard payload: %q", m.View())
	}
	if !strings.Contains(m.View(), "ДИАГНОСТИКА СКОПИРОВАНА") {
		t.Fatalf("missing copy confirmation:\n%s", m.View())
	}
	cleanup := cmd()
	updated, _ = m.Update(cleanup)
	if updated.(Model).clipboard != "" {
		t.Fatal("OSC52 pulse must be removed after it was rendered")
	}
}

func TestDiagnosticBundleSanitizesContextAndBoundsLogTail(t *testing.T) {
	t.Setenv("TERM", "xterm-256color\x1b]52;c;evil\a")
	t.Setenv("TMUX", "")
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "log line"
	}
	report := diagnosticBundle(
		Snapshot{EngineOK: true, Network: NetworkOK, Peers: []PeerRow{{DisplayName: "НЕ КОПИРОВАТЬ"}}},
		"http://192.168.1.4:17880",
		"bad\x00error\x1b",
		time.Date(2026, 7, 29, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 29, 10, 1, 0, 0, time.UTC),
		strings.Join(lines, "\n"),
	)
	if strings.ContainsAny(report, "\x00\x1b\a") {
		t.Fatalf("control character leaked into report: %q", report)
	}
	if strings.Contains(report, "192.168.1.4") || strings.Contains(report, "НЕ КОПИРОВАТЬ") {
		t.Fatalf("address or identity leaked into report:\n%s", report)
	}
	if got := strings.Count(report, "log line"); got != diagnosticLogLines {
		t.Fatalf("log lines = %d, want %d", got, diagnosticLogLines)
	}
}

func osc52Payload(t *testing.T, view string) string {
	t.Helper()
	const prefix = "\x1b]52;c;"
	start := strings.Index(view, prefix)
	if start < 0 {
		t.Fatalf("OSC52 prefix not found in %q", view)
	}
	rest := view[start+len(prefix):]
	end := strings.IndexByte(rest, '\a')
	if end < 0 {
		t.Fatalf("OSC52 terminator not found in %q", view)
	}
	return rest[:end]
}

func TestF5WithoutErrorDoesNothing(t *testing.T) {
	t.Parallel()
	m := NewModel(NewClient("http://127.0.0.1:9"))
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyF5})
	if cmd != nil {
		t.Fatal("F5 without an error must not emit a command")
	}
	if updated.(Model).clipboard != "" {
		t.Fatal("F5 without an error must not emit OSC52")
	}
}

func TestClipboardPulseOutlivesOneRefreshFrame(t *testing.T) {
	t.Parallel()
	if clipboardPulseDuration <= tickInterval {
		t.Fatalf("clipboard pulse %s must outlive refresh frame %s", clipboardPulseDuration, tickInterval)
	}
}
