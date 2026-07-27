package tui_test

import (
	"strings"
	"testing"

	"dudka/internal/tui"
)

func TestRenderCancelledTransferNotSuccess(t *testing.T) {
	t.Parallel()
	out := tui.Render(tui.Snapshot{
		MeName:   "Вася",
		EngineOK: true,
		Messages: []tui.MsgRow{{
			Type: tui.MsgTypeFileAnnounce, FileID: "f1", FileName: "a.bin", Size: 100, Mime: "application/octet-stream",
		}},
		Transfers: []tui.TransferRow{{
			FileID: "f1", Percent: 50, Status: tui.TransferCancelled,
		}},
	})
	if !strings.Contains(out, "CANCELLED") && !strings.Contains(out, "discarded") {
		t.Fatalf("want cancelled/discarded marker:\n%s", out)
	}
	if strings.Contains(out, "100%") {
		t.Fatalf("cancelled must not show 100%% success:\n%s", out)
	}
}

func TestParseCancelCommand(t *testing.T) {
	t.Parallel()
	id, ok, err := tui.ParseCancelCommand("/cancel abc-123")
	if err != nil || !ok || id != "abc-123" {
		t.Fatalf("id=%q ok=%v err=%v", id, ok, err)
	}
	_, ok, err = tui.ParseCancelCommand("/cancel")
	if !ok || err == nil {
		t.Fatal("empty cancel must error")
	}
}
