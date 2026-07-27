package tui_test

import (
	"strings"
	"testing"
	"time"

	"dudka/internal/tui"
)

func TestRenderShowsThumbMarkAndPathForImage(t *testing.T) {
	t.Parallel()
	out := tui.Render(tui.Snapshot{
		MeName:   "Вася",
		EngineOK: true,
		Network:  tui.NetworkOK,
		Messages: []tui.MsgRow{{
			DisplayName: "Аня",
			TS:          time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC),
			Type:        tui.MsgTypeFileAnnounce,
			FileID:      "fid-img",
			FileName:    "photo.jpg",
			Size:        1024,
			Mime:        "image/jpeg",
			Hash:        "sha256:aa",
			ThumbPath:   "/tmp/thumbs/fid-img.jpg",
		}},
	})
	if !strings.Contains(out, "THUMB") {
		t.Fatalf("want ASCII THUMB mark:\n%s", out)
	}
	if !strings.Contains(out, "/tmp/thumbs/fid-img.jpg") {
		t.Fatalf("want thumb path in feed:\n%s", out)
	}
}

func TestRenderNoThumbForNonImage(t *testing.T) {
	t.Parallel()
	out := tui.Render(tui.Snapshot{
		MeName:   "Вася",
		EngineOK: true,
		Network:  tui.NetworkOK,
		Messages: []tui.MsgRow{{
			DisplayName: "Аня",
			TS:          time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC),
			Type:        tui.MsgTypeFileAnnounce,
			FileID:      "fid-bin",
			FileName:    "a.bin",
			Size:        10,
			Mime:        "application/octet-stream",
			Hash:        "sha256:x",
		}},
	})
	if strings.Contains(out, "THUMB") {
		t.Fatalf("non-image must not show THUMB:\n%s", out)
	}
}
