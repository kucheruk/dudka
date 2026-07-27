package tui_test

import (
	"strings"
	"testing"
	"time"

	"dudka/internal/tui"
)

func TestRenderHEICFallbackWithoutFakeThumb(t *testing.T) {
	t.Parallel()
	out := tui.Render(tui.Snapshot{
		MeName:   "Вася",
		EngineOK: true,
		Network:  tui.NetworkOK,
		Messages: []tui.MsgRow{{
			DisplayName: "Аня",
			TS:          time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC),
			Type:        tui.MsgTypeFileAnnounce,
			FileID:      "fid-heic",
			FileName:    "img.heic",
			Size:        2048,
			Mime:        "image/heic",
			Hash:        "sha256:aa",
			// no ThumbPath — honest fallback
		}},
	})
	if !strings.Contains(out, "HEIC") {
		t.Fatalf("want honest HEIC mark:\n%s", out)
	}
	if strings.Contains(out, "ПРЕВЬЮ") {
		t.Fatalf("must not invent ПРЕВЬЮ without path:\n%s", out)
	}
}

func TestRenderHEICWithThumbShowsThumb(t *testing.T) {
	t.Parallel()
	out := tui.Render(tui.Snapshot{
		MeName:   "Вася",
		EngineOK: true,
		Network:  tui.NetworkOK,
		Messages: []tui.MsgRow{{
			DisplayName: "Аня",
			TS:          time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC),
			Type:        tui.MsgTypeFileAnnounce,
			FileID:      "fid-heic2",
			FileName:    "img.heic",
			Size:        2048,
			Mime:        "image/heic",
			Hash:        "sha256:aa",
			ThumbPath:   "/tmp/thumbs/fid-heic2.jpg",
		}},
	})
	if !strings.Contains(out, "ПРЕВЬЮ /tmp/thumbs/fid-heic2.jpg") {
		t.Fatalf("want ПРЕВЬЮ path when present:\n%s", out)
	}
}
