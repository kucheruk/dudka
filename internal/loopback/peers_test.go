package loopback_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dudka/internal/discovery"
	"dudka/internal/loopback"
)

func TestGetPeersReturnsNeighbors(t *testing.T) {
	t.Parallel()
	store := discovery.NewPeerStore()
	store.Upsert(discovery.Peer{
		PeerID: "neighbor", DisplayName: "Боб", InstanceID: "i",
		Host: "192.168.1.5", TCPPort: 41777, LastSeen: time.Now(),
	})
	srv := loopback.New("me", "Я")
	srv.SetPeers(store)

	req := httptest.NewRequest(http.MethodGet, "/peers", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Peers []struct {
			PeerID      string `json:"peer_id"`
			DisplayName string `json:"display_name"`
		} `json:"peers"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Peers) != 1 || body.Peers[0].PeerID != "neighbor" || body.Peers[0].DisplayName != "Боб" {
		t.Fatalf("body=%+v", body)
	}
}
