package loopback

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInternetConsentIsExplicit(t *testing.T) {
	enabled := false
	server := New("peer", "Имя")
	server.SetInternetConsent(
		func() bool { return enabled },
		func() error {
			enabled = true
			return nil
		},
	)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	assertConsent := func(want bool) {
		t.Helper()
		response, err := http.Get(httpServer.URL + "/internet-consent")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var payload struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Enabled != want {
			t.Fatalf("enabled=%v, want %v", payload.Enabled, want)
		}
	}
	assertConsent(false)
	response, err := http.Post(httpServer.URL+"/internet-consent", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("POST status=%d", response.StatusCode)
	}
	assertConsent(true)
}
