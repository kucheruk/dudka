package tui_test

import (
	"testing"

	"dudka/internal/tui"
)

func TestParseAnnounceCommand(t *testing.T) {
	t.Parallel()
	path, ok, err := tui.ParseAnnounceCommand("/announce /tmp/a.jpg")
	if err != nil || !ok || path != "/tmp/a.jpg" {
		t.Fatalf("path=%q ok=%v err=%v", path, ok, err)
	}
	_, ok, err = tui.ParseAnnounceCommand("/announce")
	if !ok || err == nil {
		t.Fatal("want error for missing path")
	}
	_, ok, err = tui.ParseAnnounceCommand("hello")
	if ok || err != nil {
		t.Fatalf("not a command: ok=%v err=%v", ok, err)
	}
}
