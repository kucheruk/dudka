package tui

import (
	"encoding/base64"
	"strings"
	"testing"

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
	if !strings.Contains(m.View(), "F5 · КОПИРОВАТЬ ОШИБКУ") {
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
	wantPayload := base64.StdEncoding.EncodeToString([]byte("поиск: context deadline exceeded"))
	if !strings.Contains(m.View(), "\x1b]52;c;"+wantPayload+"\x07") {
		t.Fatalf("view does not contain OSC52 clipboard payload: %q", m.View())
	}
	if !strings.Contains(m.View(), "ОШИБКА СКОПИРОВАНА") {
		t.Fatalf("missing copy confirmation:\n%s", m.View())
	}
	cleanup := cmd()
	updated, _ = m.Update(cleanup)
	if updated.(Model).clipboard != "" {
		t.Fatal("OSC52 pulse must be removed after it was rendered")
	}
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
