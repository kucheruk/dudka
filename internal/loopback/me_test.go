package loopback_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"dudka/internal/loopback"
)

func TestMeReturnsPeerJSON(t *testing.T) {
	t.Parallel()
	srv := loopback.New("peer-aaa", "Вася")
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", ct)
	}
	var body struct {
		PeerID string `json:"peer_id"`
		Name   string `json:"name"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v body=%s", err, rr.Body.String())
	}
	if body.PeerID != "peer-aaa" || body.Name != "Вася" {
		t.Fatalf("body = %+v", body)
	}
}

func TestMeRejectsNonLoopbackRemote(t *testing.T) {
	t.Parallel()
	srv := loopback.New("peer-aaa", "Вася")
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.RemoteAddr = "203.0.113.10:9999"
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
}

func TestHealthAlsoRejectsNonLoopbackRemote(t *testing.T) {
	t.Parallel()
	srv := loopback.New("peer-aaa", "Вася")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "10.0.0.5:1"
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
}
