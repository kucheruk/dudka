package tui_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dudka/internal/tui"
)

func TestIsLargeFileThreshold(t *testing.T) {
	t.Parallel()
	if tui.IsLargeFile(tui.LargeFileBytes) {
		t.Fatal("exactly 100 MiB must not warn")
	}
	if !tui.IsLargeFile(tui.LargeFileBytes + 1) {
		t.Fatal("101 MiB-ish must warn")
	}
	if tui.IsLargeFile(0) || tui.IsLargeFile(1024) {
		t.Fatal("small files must not warn")
	}
}

func TestBeginFetchWarnsBeforeStartWhenLarge(t *testing.T) {
	t.Parallel()
	var fetchHits int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /me", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"peer_id": "p1", "name": "A"})
	})
	mux.HandleFunc("GET /peers", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"peers": []any{}})
	})
	mux.HandleFunc("GET /status", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"proto_major": 1, "proto_minor": 0, "network": "ok"})
	})
	mux.HandleFunc("GET /messages", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]any{{
				"type": "file_announce", "file_id": "big-1", "name": "huge.bin",
				"size": tui.LargeFileBytes + 1, "mime": "application/octet-stream", "hash": "sha256:x",
				"display_name_at_send": "A",
			}},
		})
	})
	mux.HandleFunc("GET /files/transfers", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"transfers": []any{}})
	})
	mux.HandleFunc("POST /files/fetch", func(w http.ResponseWriter, _ *http.Request) {
		fetchHits++
		_ = json.NewEncoder(w).Encode(map[string]any{"file_id": "big-1", "status": "downloading", "percent": 0})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c := tui.NewClient(srv.URL)

	plan, err := c.BeginFetch("big-1", false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Warning == "" || !strings.Contains(plan.Warning, "100") {
		t.Fatalf("want large-file warning, got %+v", plan)
	}
	if plan.Started || fetchHits != 0 {
		t.Fatalf("must not start fetch before confirm; hits=%d plan=%+v", fetchHits, plan)
	}

	plan2, err := c.BeginFetch("big-1", true)
	if err != nil {
		t.Fatal(err)
	}
	if plan2.Warning != "" {
		t.Fatalf("force must not re-warn: %+v", plan2)
	}
	if !plan2.Started || fetchHits != 1 {
		t.Fatalf("force must start fetch; hits=%d plan=%+v", fetchHits, plan2)
	}
}

func TestBeginFetchSmallStartsImmediately(t *testing.T) {
	t.Parallel()
	var fetchHits int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /me", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"peer_id": "p1", "name": "A"})
	})
	mux.HandleFunc("GET /peers", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"peers": []any{}})
	})
	mux.HandleFunc("GET /status", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"proto_major": 1, "network": "ok"})
	})
	mux.HandleFunc("GET /messages", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]any{{
				"type": "file_announce", "file_id": "small-1", "name": "a.txt",
				"size": 10, "mime": "text/plain", "hash": "sha256:x",
			}},
		})
	})
	mux.HandleFunc("GET /files/transfers", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"transfers": []any{}})
	})
	mux.HandleFunc("POST /files/fetch", func(w http.ResponseWriter, _ *http.Request) {
		fetchHits++
		_ = json.NewEncoder(w).Encode(map[string]any{"file_id": "small-1", "status": "downloading"})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	plan, err := tui.NewClient(srv.URL).BeginFetch("small-1", false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Warning != "" || !plan.Started || fetchHits != 1 {
		t.Fatalf("small file should start; plan=%+v hits=%d", plan, fetchHits)
	}
}

func TestParseFetchCommandForce(t *testing.T) {
	t.Parallel()
	id, force, ok, err := tui.ParseFetchCommand("/fetch! abc")
	if err != nil || !ok || !force || id != "abc" {
		t.Fatalf("id=%q force=%v ok=%v err=%v", id, force, ok, err)
	}
	id, force, ok, err = tui.ParseFetchCommand("/fetch abc --yes")
	if err != nil || !ok || !force || id != "abc" {
		t.Fatalf("yes: id=%q force=%v ok=%v err=%v", id, force, ok, err)
	}
	id, force, ok, err = tui.ParseFetchCommand("/fetch abc")
	if err != nil || !ok || force || id != "abc" {
		t.Fatalf("plain: id=%q force=%v ok=%v err=%v", id, force, ok, err)
	}
}

func TestRenderLargeFileShowsWarn(t *testing.T) {
	t.Parallel()
	out := tui.Render(tui.Snapshot{
		MeName:   "Вася",
		EngineOK: true,
		Messages: []tui.MsgRow{{
			Type: tui.MsgTypeFileAnnounce, FileID: "f", FileName: "huge.bin",
			Size: tui.LargeFileBytes + 1, Mime: "application/octet-stream",
		}},
	})
	if !strings.Contains(out, "WARN") || !strings.Contains(out, "100") {
		t.Fatalf("want WARN>100MiB marker:\n%s", out)
	}
}
