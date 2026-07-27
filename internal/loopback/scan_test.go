package loopback_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"dudka/internal/discovery"
	"dudka/internal/loopback"
)

func TestPostScanInvokesProvider(t *testing.T) {
	t.Parallel()
	srv := loopback.New("me", "Я")
	called := false
	srv.SetScanProvider(func(ctx context.Context, req discovery.ScanRequest) (discovery.ScanResult, error) {
		called = true
		if len(req.Hosts) != 1 || req.Hosts[0] != "127.0.0.1" || req.Port != 9 {
			t.Fatalf("req=%+v", req)
		}
		return discovery.ScanResult{Probed: 1, Found: 1, Peers: []discovery.Peer{{PeerID: "x"}}}, nil
	})

	body := bytes.NewBufferString(`{"hosts":["127.0.0.1"],"port":9}`)
	req := httptest.NewRequest(http.MethodPost, "/scan", body)
	req.RemoteAddr = "127.0.0.1:1"
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !called {
		t.Fatal("scan provider not called")
	}
	var res discovery.ScanResult
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Found != 1 {
		t.Fatalf("res=%+v", res)
	}
}
