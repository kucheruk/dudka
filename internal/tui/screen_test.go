package tui_test

import (
	"strings"
	"testing"
	"time"

	"dudka/internal/tui"
)

func TestRenderScreenHasRussianPanels(t *testing.T) {
	t.Parallel()
	out := tui.RenderScreen(tui.ScreenState{
		Snap: tui.Snapshot{
			MeName:     "Аня",
			EngineOK:   true,
			Network:    tui.NetworkOK,
			ProtoMajor: 1,
			Peers:      []tui.PeerRow{{PeerID: "p2", DisplayName: "Боря"}},
			Messages: []tui.MsgRow{{
				DisplayName: "Боря",
				Text:        "привет",
				TS:          time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC),
				Type:        tui.MsgTypeChat,
			}},
		},
		Compose:  "черновик",
		CursorOn: true,
	}, 80, 24)
	for _, want := range []string{"С О С Е Д И", "Л Е Н Т А", "ОТПРАВИТЬ", "онлайн 2", "Аня · ВЫ", "Боря", "привет", "черновик"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	for _, ban := range []string{"\nPEERS\n", "\nFEED\n", "online 1", "ENGINE OFFLINE", "ДУНУТЬ", "дунуть"} {
		if strings.Contains(out, ban) {
			t.Fatalf("banned %q in:\n%s", ban, out)
		}
	}
	// Truecolor charcoal background sequences must be present (not plain B&W dump).
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected ANSI color escapes in screen render")
	}
}

func TestRenderScreenAloneShowsSeek(t *testing.T) {
	t.Parallel()
	out := tui.RenderScreen(tui.ScreenState{
		Snap: tui.Snapshot{MeName: "Аня", EngineOK: true, Network: tui.NetworkOK},
	}, 80, 20)
	if !strings.Contains(out, tui.EmptyPeersCopy) {
		t.Fatalf("missing alone copy:\n%s", out)
	}
	if !strings.Contains(out, "ИСКАТЬ") {
		t.Fatalf("missing seek affordance:\n%s", out)
	}
	if !strings.Contains(out, "/search") {
		t.Fatalf("missing explicit search command:\n%s", out)
	}
}

func TestRenderScreenErrorShowsCopyAction(t *testing.T) {
	t.Parallel()
	out := tui.RenderScreen(tui.ScreenState{
		Snap:         tui.Snapshot{MeName: "Аня", EngineOK: true, Network: tui.NetworkOK},
		StatusMsg:    "ПОИСК НЕ ЗАВЕРШЁН · повторите /search",
		StatusError:  true,
		CanCopyError: true,
	}, 100, 20)
	if !strings.Contains(out, "F5 · КОПИРОВАТЬ ОШИБКУ") {
		t.Fatalf("missing copy-error action:\n%s", out)
	}
}

func TestNewModelInitAndViewSmoke(t *testing.T) {
	t.Parallel()
	m := tui.NewModel(tui.NewClient("http://127.0.0.1:9"))
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init must return cmds")
	}
	view := m.View()
	if !strings.Contains(view, "ДУДКА") || !strings.Contains(view, "С О С Е Д И") {
		t.Fatalf("view missing panels:\n%s", view)
	}
}
