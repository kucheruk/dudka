package tui_test

import (
	"strings"
	"testing"
	"time"

	"dudka/internal/tui"
)

func TestRenderFileAnnounceInFeed(t *testing.T) {
	t.Parallel()
	out := tui.Render(tui.Snapshot{
		MeName:   "Вася",
		EngineOK: true,
		Network:  tui.NetworkOK,
		Messages: []tui.MsgRow{{
			DisplayName: "Аня",
			TS:          time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC),
			Type:        tui.MsgTypeFileAnnounce,
			FileID:      "fid-1",
			FileName:    "photo.jpg",
			Size:        1024,
			Mime:        "image/jpeg",
			Hash:        "sha256:aa",
		}},
	})
	for _, part := range []string{"ЛЕНТА", "Аня", "photo.jpg", "1024", "image/jpeg", "fid-1"} {
		if !strings.Contains(out, part) {
			t.Fatalf("missing %q in:\n%s", part, out)
		}
	}
	if strings.Contains(out, "ФАЙЛ BODY") {
		t.Fatalf("must not invent file body:\n%s", out)
	}
}
