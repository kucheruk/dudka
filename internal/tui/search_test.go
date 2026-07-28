package tui

import "testing"

func TestParseSearchCommand(t *testing.T) {
	t.Parallel()
	for _, command := range []string{"/search", "/SEARCH", "/scan", "/поиск"} {
		if !ParseSearchCommand(command) {
			t.Fatalf("%q must be a search command", command)
		}
	}
	for _, text := range []string{"s", "S", "Search", "сообщение с s"} {
		if ParseSearchCommand(text) {
			t.Fatalf("%q must remain ordinary message text", text)
		}
	}
}
