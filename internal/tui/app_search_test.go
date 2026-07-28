package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSRemainsComposeText(t *testing.T) {
	t.Parallel()
	m := NewModel(NewClient("http://127.0.0.1:9"))
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	got := updated.(Model)
	if got.input.Value() != "S" {
		t.Fatalf("input=%q want ordinary S", got.input.Value())
	}
}
