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

func TestClientSetNick(t *testing.T) {
	t.Parallel()
	var gotBody string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /nick", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		_ = json.NewEncoder(w).Encode(map[string]string{"peer_id": "p1", "name": "Новый"})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := tui.NewClient(srv.URL)
	name, err := c.SetNick("Новый")
	if err != nil {
		t.Fatal(err)
	}
	if name != "Новый" {
		t.Fatalf("name=%q", name)
	}
	if !strings.Contains(gotBody, "Новый") {
		t.Fatalf("body=%s", gotBody)
	}
}

func TestHandleComposeLineNickCommand(t *testing.T) {
	t.Parallel()
	var nick string
	var sent string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /nick", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		nick = req.Name
		_ = json.NewEncoder(w).Encode(map[string]string{"peer_id": "p", "name": req.Name})
	})
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
	if err := tui.HandleComposeLine(c, "/nick Катя"); err != nil {
		t.Fatal(err)
	}
	if nick != "Катя" {
		t.Fatalf("nick=%q", nick)
	}
	if sent != "" {
		t.Fatalf("nick command must not send chat text, sent=%q", sent)
	}
}

func TestNickAppliesToSubsequentSend(t *testing.T) {
	t.Parallel()
	var current = "Старый"
	var atSend string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /nick", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		current = req.Name
		_ = json.NewEncoder(w).Encode(map[string]string{"peer_id": "p", "name": current})
	})
	mux.HandleFunc("POST /send", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Text string `json:"text"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		atSend = current
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "accepted",
			"queued": 0,
			"message": map[string]any{
				"text":                 req.Text,
				"display_name_at_send": current,
			},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := tui.NewClient(srv.URL)
	if err := tui.HandleComposeLine(c, "/nick НовыйНик"); err != nil {
		t.Fatal(err)
	}
	if err := tui.HandleComposeLine(c, "после смены"); err != nil {
		t.Fatal(err)
	}
	if atSend != "НовыйНик" {
		t.Fatalf("display_name_at_send=%q want НовыйНик", atSend)
	}
}

func TestRenderMentionsNickCommand(t *testing.T) {
	t.Parallel()
	out := tui.Render(tui.Snapshot{MeName: "A", EngineOK: true})
	if !strings.Contains(out, "/nick") {
		t.Fatalf("INPUT should mention /nick:\n%s", out)
	}
}
