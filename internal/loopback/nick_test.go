package loopback_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"dudka/internal/loopback"
)

func TestPostNickUpdatesMe(t *testing.T) {
	t.Parallel()
	srv := loopback.New("peer-1", "Старый")
	var persisted atomic.Value
	srv.SetPersistName(func(name string) error {
		persisted.Store(name)
		return nil
	})

	req := httptest.NewRequest(http.MethodPost, "/nick", bytes.NewBufferString(`{"name":"Новый"}`))
	req.RemoteAddr = "127.0.0.1:1"
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /nick status=%d body=%s", rr.Code, rr.Body.String())
	}

	meReq := httptest.NewRequest(http.MethodGet, "/me", nil)
	meReq.RemoteAddr = "127.0.0.1:1"
	meRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(meRR, meReq)
	if meRR.Code != http.StatusOK {
		t.Fatalf("GET /me status=%d", meRR.Code)
	}
	var body struct {
		PeerID string `json:"peer_id"`
		Name   string `json:"name"`
	}
	if err := json.Unmarshal(meRR.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Name != "Новый" || body.PeerID != "peer-1" {
		t.Fatalf("me after nick = %+v", body)
	}
	if got, _ := persisted.Load().(string); got != "Новый" {
		t.Fatalf("persist got %q", got)
	}
}

func TestPostNickRejectsEmpty(t *testing.T) {
	t.Parallel()
	srv := loopback.New("peer-1", "Старый")
	req := httptest.NewRequest(http.MethodPost, "/nick", bytes.NewBufferString(`{"name":"  "}`))
	req.RemoteAddr = "127.0.0.1:1"
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", rr.Code)
	}
	meReq := httptest.NewRequest(http.MethodGet, "/me", nil)
	meReq.RemoteAddr = "127.0.0.1:1"
	meRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(meRR, meReq)
	var body struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(meRR.Body.Bytes(), &body)
	if body.Name != "Старый" {
		t.Fatalf("name changed on bad request: %q", body.Name)
	}
}

func TestPostNickRejectsNonLoopback(t *testing.T) {
	t.Parallel()
	srv := loopback.New("peer-1", "Старый")
	req := httptest.NewRequest(http.MethodPost, "/nick", bytes.NewBufferString(`{"name":"X"}`))
	req.RemoteAddr = "203.0.113.9:1"
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rr.Code)
	}
}
