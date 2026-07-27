package tui_test

import (
	"strings"
	"testing"
	"time"

	"dudka/internal/tui"
)

func TestRenderFileAnnounceShowsDownloadPercent(t *testing.T) {
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
		}},
		Transfers: []tui.TransferRow{{
			FileID:  "fid-1",
			Name:    "photo.jpg",
			Percent: 42,
			Status:  tui.TransferDownloading,
		}},
	})
	if !strings.Contains(out, "42%") {
		t.Fatalf("want 42%% progress in:\n%s", out)
	}
	if !strings.Contains(out, "photo.jpg") {
		t.Fatalf("missing file name:\n%s", out)
	}
}

func TestRenderDoneTransferShowsHundredPercent(t *testing.T) {
	t.Parallel()
	out := tui.Render(tui.Snapshot{
		MeName:   "Вася",
		EngineOK: true,
		Messages: []tui.MsgRow{{
			Type: tui.MsgTypeFileAnnounce, FileID: "f", FileName: "a.bin", Size: 10, Mime: "application/octet-stream",
		}},
		Transfers: []tui.TransferRow{{
			FileID: "f", Percent: 100, Status: tui.TransferDone,
		}},
	})
	if !strings.Contains(out, "100%") {
		t.Fatalf("want 100%%:\n%s", out)
	}
}
