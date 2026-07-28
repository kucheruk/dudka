package tui_test

import (
	"strings"
	"testing"
	"time"

	"dudka/internal/tui"
)

// P072 / DUD-PRD-103: user-facing TUI frame must be Russian (except paths/MIME/commands).
func TestRenderUserFacingRussian(t *testing.T) {
	t.Parallel()
	out := tui.Render(tui.Snapshot{
		MeName:     "Аня",
		EngineOK:   true,
		Network:    tui.NetworkOK,
		ProtoMajor: 1,
		ProtoMinor: 0,
		Peers:      []tui.PeerRow{{PeerID: "p1", DisplayName: "Боря"}},
		Messages: []tui.MsgRow{{
			DisplayName: "Боря",
			TS:          time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC),
			Type:        tui.MsgTypeFileAnnounce,
			FileID:      "fid-1",
			FileName:    "photo.jpg",
			Size:        1024,
			Mime:        "image/jpeg",
		}},
	})
	for _, want := range []string{"СОСЕДИ", "ЛЕНТА", "ВВОД", "онлайн 2", "Аня · ВЫ", "прото 1.0", "ФАЙЛ photo.jpg"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	for _, ban := range []string{"\nPEERS\n", "\nFEED\n", "\nINPUT\n", " online ", " proto ", "FILE photo.jpg", "ENGINE OFFLINE"} {
		if strings.Contains(out, ban) {
			t.Fatalf("banned English %q in:\n%s", ban, out)
		}
	}
}

func TestRenderOfflineAndAloneRussian(t *testing.T) {
	t.Parallel()
	offline := tui.Render(tui.Snapshot{EngineOK: false, Err: "boom"})
	if !strings.Contains(offline, "ДВИЖОК НЕДОСТУПЕН") {
		t.Fatalf("offline RU missing:\n%s", offline)
	}
	if strings.Contains(offline, "ENGINE OFFLINE") {
		t.Fatalf("English offline:\n%s", offline)
	}

	alone := tui.Render(tui.Snapshot{MeName: "Аня", EngineOK: true, Network: tui.NetworkOK})
	if !strings.Contains(alone, "один") || !strings.Contains(alone, "скан подсети") {
		t.Fatalf("alone RU missing:\n%s", alone)
	}
	if strings.Contains(alone, "subnet scan") || strings.Contains(alone, " alone") {
		t.Fatalf("English alone tokens:\n%s", alone)
	}

	noNet := tui.Render(tui.Snapshot{MeName: "Аня", EngineOK: true, Network: tui.NetworkNoNetwork})
	if !strings.Contains(noNet, "нет сети") {
		t.Fatalf("no_network RU missing:\n%s", noNet)
	}
	if strings.Contains(noNet, "no_network") {
		t.Fatalf("raw no_network token still shown:\n%s", noNet)
	}
}

func TestRenderTransferMarkersRussian(t *testing.T) {
	t.Parallel()
	cancelled := tui.Render(tui.Snapshot{
		MeName: "Вася", EngineOK: true,
		Messages: []tui.MsgRow{{
			Type: tui.MsgTypeFileAnnounce, FileID: "f1", FileName: "a.bin", Size: 100, Mime: "application/octet-stream",
		}},
		Transfers: []tui.TransferRow{{FileID: "f1", Percent: 50, Status: tui.TransferCancelled}},
	})
	if !strings.Contains(cancelled, "ОТМЕНЕНО") {
		t.Fatalf("want ОТМЕНЕНО:\n%s", cancelled)
	}
	if strings.Contains(cancelled, "CANCELLED") || strings.Contains(cancelled, "discarded") {
		t.Fatalf("English cancel marker:\n%s", cancelled)
	}

	errFrame := tui.Render(tui.Snapshot{
		MeName: "Вася", EngineOK: true,
		Messages: []tui.MsgRow{{
			Type: tui.MsgTypeFileAnnounce, FileID: "f2", FileName: "b.bin", Size: 10, Mime: "application/octet-stream",
		}},
		Transfers: []tui.TransferRow{{FileID: "f2", Status: tui.TransferError}},
	})
	if !strings.Contains(errFrame, "ОШИБКА") || strings.Contains(errFrame, " ERROR") {
		t.Fatalf("want ОШИБКА:\n%s", errFrame)
	}

	large := tui.Render(tui.Snapshot{
		MeName: "Вася", EngineOK: true,
		Messages: []tui.MsgRow{{
			Type: tui.MsgTypeFileAnnounce, FileID: "f", FileName: "huge.bin",
			Size: tui.LargeFileBytes + 1, Mime: "application/octet-stream",
		}},
	})
	if !strings.Contains(large, "ВНИМАНИЕ") || strings.Contains(large, "WARN") {
		t.Fatalf("want ВНИМАНИЕ marker:\n%s", large)
	}
}
