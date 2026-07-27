package tui_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dudka/internal/tui"
)

func TestFetchSnapshotFromEngine(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /me", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"peer_id": "p1", "name": "Аня"})
	})
	mux.HandleFunc("GET /peers", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"peers": []map[string]any{
				{"peer_id": "p2", "display_name": "Боря"},
			},
		})
	})
	mux.HandleFunc("GET /status", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"proto_major": 1, "proto_minor": 0})
	})
	mux.HandleFunc("GET /messages", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]any{
				{
					"msg_id":               "m1",
					"peer_id":              "p1",
					"display_name_at_send": "Аня",
					"ts":                   time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC),
					"text":                 "из engine",
				},
			},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := tui.NewClient(srv.URL)
	snap, err := c.Fetch()
	if err != nil {
		t.Fatal(err)
	}
	if !snap.EngineOK || snap.MeName != "Аня" || snap.PeerID != "p1" {
		t.Fatalf("snap=%+v", snap)
	}
	if len(snap.Peers) != 1 || snap.Peers[0].DisplayName != "Боря" {
		t.Fatalf("peers=%+v", snap.Peers)
	}
	if snap.ProtoMajor != 1 {
		t.Fatalf("proto=%d", snap.ProtoMajor)
	}
	if len(snap.Messages) != 1 || snap.Messages[0].Text != "из engine" {
		t.Fatalf("messages=%+v", snap.Messages)
	}
	frame := tui.Render(snap)
	for _, part := range []string{"Аня", "Боря", "online 1", "FEED", "из engine"} {
		if !strings.Contains(frame, part) {
			t.Fatalf("missing %q in:\n%s", part, frame)
		}
	}
}
