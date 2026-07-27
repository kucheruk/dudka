package tui_test

import (
	"strings"
	"testing"
	"time"

	"dudka/internal/tui"
)

func TestRenderShowsMessageFeed(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, 7, 27, 15, 4, 0, 0, time.UTC)
	out := tui.Render(tui.Snapshot{
		MeName:   "Вася",
		EngineOK: true,
		Peers:    nil, // alone — feed still visible (P041)
		Messages: []tui.MsgRow{
			{DisplayName: "Вася", Text: "привет из комнаты", TS: ts},
			{DisplayName: "Боря", Text: "отвечаю", TS: ts.Add(time.Minute)},
		},
	})
	if !strings.Contains(out, "ЛЕНТА") {
		t.Fatalf("missing ЛЕНТА:\n%s", out)
	}
	if !strings.Contains(out, tui.EmptyPeersCopy) {
		t.Fatalf("alone should still show empty peers copy:\n%s", out)
	}
	if !strings.Contains(out, "привет из комнаты") || !strings.Contains(out, "отвечаю") {
		t.Fatalf("missing message text:\n%s", out)
	}
	if !strings.Contains(out, "Вася") || !strings.Contains(out, "Боря") {
		t.Fatalf("missing nicks in feed:\n%s", out)
	}
	// DESIGN: время · ник · текст
	if !strings.Contains(out, "15:04") || !strings.Contains(out, "·") {
		t.Fatalf("want time · nick · text row:\n%s", out)
	}
}

func TestRenderEmptyFeedSection(t *testing.T) {
	t.Parallel()
	out := tui.Render(tui.Snapshot{
		MeName:   "Вася",
		EngineOK: true,
		Messages: nil,
	})
	if !strings.Contains(out, "ЛЕНТА") {
		t.Fatalf("missing ЛЕНТА header:\n%s", out)
	}
}
