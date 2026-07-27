package loopback_test

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"dudka/internal/loopback"
)

func TestHealthReturns200(t *testing.T) {
	t.Parallel()
	srv := loopback.New("peer-health", "Health")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ln) }()

	url := fmt.Sprintf("http://%s/health", ln.Addr().String())
	client := &http.Client{Timeout: 2 * time.Second}
	var resp *http.Response
	deadline := time.Now().Add(2 * time.Second)
	for {
		resp, err = client.Get(url)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("GET /health: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok\n" {
		t.Fatalf("body = %q, want ok\\n", body)
	}

	_ = ln.Close()
	select {
	case <-errCh:
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return after listener close")
	}
}

func TestListenRejectsNonLoopback(t *testing.T) {
	t.Parallel()
	srv := loopback.New("peer-health", "Health")
	_, err := srv.Listen("0.0.0.0:0")
	if err == nil {
		t.Fatal("expected error for non-loopback listen addr")
	}
}

func TestFormatReady(t *testing.T) {
	t.Parallel()
	got := loopback.FormatReady("abc-123", "Вася")
	want := "ready peer_id=abc-123 name=Вася"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
