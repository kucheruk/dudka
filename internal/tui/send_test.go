package tui_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dudka/internal/tui"
)

func TestClientSendPostsText(t *testing.T) {
	t.Parallel()
	var gotBody string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /send", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "accepted",
			"queued":  0,
			"message": map[string]any{"text": "ping", "msg_id": "m1"},
		})
	})
	// Fetch endpoints unused here.
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := tui.NewClient(srv.URL)
	res, err := c.Send("ping")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "accepted" && res.Status != "queued" {
		t.Fatalf("status=%q", res.Status)
	}
	if !strings.Contains(gotBody, `"text":"ping"`) && !strings.Contains(gotBody, `"text": "ping"`) {
		t.Fatalf("body=%s", gotBody)
	}
}

func TestClientSendRejectsEmpty(t *testing.T) {
	t.Parallel()
	c := tui.NewClient("http://127.0.0.1:1")
	_, err := c.Send("   ")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestHandleComposeLineSends(t *testing.T) {
	t.Parallel()
	var sent string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /send", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Text string `json:"text"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		sent = req.Text
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "accepted", "queued": 0})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := tui.NewClient(srv.URL)
	if err := tui.HandleComposeLine(c, "привет\n"); err != nil {
		t.Fatal(err)
	}
	if sent != "привет" {
		t.Fatalf("sent=%q", sent)
	}
	if err := tui.HandleComposeLine(c, "   \n"); err != nil {
		t.Fatalf("blank line should be no-op, got %v", err)
	}
}

func TestRenderShowsInputHint(t *testing.T) {
	t.Parallel()
	out := tui.Render(tui.Snapshot{MeName: "A", EngineOK: true})
	if !strings.Contains(out, "INPUT") {
		t.Fatalf("missing INPUT:\n%s", out)
	}
}
