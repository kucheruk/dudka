package tui

import (
	"os"
	"strings"

	"github.com/aymanbagabas/go-osc52/v2"
)

// osc52Sequence asks the user's terminal to copy text to its system clipboard.
// It needs no Linux desktop packages and works through SSH because the terminal
// that owns the clipboard interprets the escape sequence.
func osc52Sequence(text string) string {
	seq := osc52.New(text)
	switch {
	case os.Getenv("TMUX") != "":
		seq = seq.Tmux()
	case strings.HasPrefix(os.Getenv("TERM"), "screen"):
		seq = seq.Screen()
	}
	return seq.String()
}
