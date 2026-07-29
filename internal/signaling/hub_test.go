package signaling

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestSameIPPeersMeetAndRouteOnlySignaling(t *testing.T) {
	t.Parallel()
	server := NewServer("http://example.test")
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	first := dialPeer(t, httpServer.URL, "198.51.100.20")
	firstWelcome := readWire(t, first)
	if firstWelcome.Type != "welcome" || len(firstWelcome.Peers) != 0 {
		t.Fatalf("first welcome = %+v", firstWelcome)
	}

	second := dialPeer(t, httpServer.URL, "198.51.100.20")
	secondWelcome := readWire(t, second)
	if len(secondWelcome.Peers) != 1 || secondWelcome.Peers[0] != firstWelcome.From {
		t.Fatalf("second welcome = %+v, first = %+v", secondWelcome, firstWelcome)
	}

	offer := wireSignal{
		Type:        "offer",
		To:          firstWelcome.From,
		Description: json.RawMessage(`{"type":"offer","sdp":"v=0"}`),
	}
	writeWire(t, second, offer)
	routed := readWire(t, first)
	if routed.From != secondWelcome.From || routed.To != firstWelcome.From || routed.Type != "offer" {
		t.Fatalf("routed = %+v", routed)
	}

	writeWire(t, second, wireSignal{
		Type:        "chat",
		To:          firstWelcome.From,
		Description: json.RawMessage(`{"text":"не должно пройти"}`),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _, err := second.Read(ctx)
	if websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
		t.Fatalf("chat signaling close status = %v, err=%v", websocket.CloseStatus(err), err)
	}
}

func TestDifferentIPRoomsAreIsolated(t *testing.T) {
	t.Parallel()
	server := NewServer("http://example.test")
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	first := dialPeer(t, httpServer.URL, "198.51.100.21")
	firstWelcome := readWire(t, first)
	second := dialPeer(t, httpServer.URL, "198.51.100.22")
	secondWelcome := readWire(t, second)
	if len(secondWelcome.Peers) != 0 {
		t.Fatalf("different IP leaked peers: %+v vs %+v", firstWelcome, secondWelcome)
	}
}

func TestForeignOriginRejected(t *testing.T) {
	t.Parallel()
	server := NewServer("http://example.test")
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	headers := http.Header{"Origin": []string{"https://evil.example"}}
	_, resp, err := websocket.Dial(ctx, wsURL(httpServer.URL), &websocket.DialOptions{HTTPHeader: headers})
	if err == nil || resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign origin: resp=%v err=%v", resp, err)
	}
}

func TestRoomRemovedAfterLastPeerLeaves(t *testing.T) {
	t.Parallel()
	server := NewServer("http://example.test")
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	peer := dialPeer(t, httpServer.URL, "198.51.100.23")
	_ = readWire(t, peer)
	_ = peer.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		server.mu.Lock()
		rooms := len(server.rooms)
		server.mu.Unlock()
		if rooms == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("room retained after disconnect")
}

func TestDebugExposesOnlyAggregateCounts(t *testing.T) {
	t.Parallel()
	server := NewServer("http://example.test")
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	first := dialPeer(t, httpServer.URL, "198.51.100.30")
	_ = readWire(t, first)
	second := dialPeer(t, httpServer.URL, "198.51.100.30")
	_ = readWire(t, second)

	response, err := http.Get(httpServer.URL + "/debug")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var snapshot struct {
		Rooms     int   `json:"rooms"`
		Peers     int   `json:"peers"`
		RoomSizes []int `json:"room_sizes"`
	}
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Rooms != 1 || snapshot.Peers != 2 ||
		len(snapshot.RoomSizes) != 1 || snapshot.RoomSizes[0] != 2 {
		t.Fatalf("debug snapshot = %+v", snapshot)
	}
}

func dialPeer(t *testing.T, baseURL, forwardedIP string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	headers := http.Header{
		"Origin":          []string{"http://example.test"},
		"X-Forwarded-For": []string{forwardedIP},
	}
	conn, _, err := websocket.Dial(ctx, wsURL(baseURL), &websocket.DialOptions{HTTPHeader: headers})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.CloseNow() })
	return conn
}

func readWire(t *testing.T, conn *websocket.Conn) wireSignal {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, raw, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var msg wireSignal
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
	return msg
}

func writeWire(t *testing.T, conn *websocket.Conn, msg wireSignal) {
	t.Helper()
	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, raw); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func wsURL(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http")
}
