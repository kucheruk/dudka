package tui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client talks to dudkad loopback HTTP.
type Client struct {
	base   string
	client *http.Client
}

// NewClient builds a client for engine base URL (e.g. http://127.0.0.1:17880).
func NewClient(baseURL string) *Client {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base != "" && !strings.Contains(base, "://") {
		base = "http://" + base
	}
	return &Client{
		base: base,
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

// Fetch loads /me, /peers, /status, /messages into a Snapshot.
func (c *Client) Fetch() (Snapshot, error) {
	var snap Snapshot
	if c.base == "" {
		return Snapshot{EngineOK: false, Err: "engine URL empty"}, fmt.Errorf("tui: engine URL empty")
	}

	var me struct {
		PeerID string `json:"peer_id"`
		Name   string `json:"name"`
	}
	if err := c.getJSON("/me", &me); err != nil {
		return Snapshot{EngineOK: false, Err: err.Error()}, err
	}
	snap.PeerID = me.PeerID
	snap.MeName = me.Name

	var peersEnv struct {
		Peers []struct {
			PeerID      string `json:"peer_id"`
			DisplayName string `json:"display_name"`
		} `json:"peers"`
	}
	if err := c.getJSON("/peers", &peersEnv); err != nil {
		return Snapshot{EngineOK: false, Err: err.Error()}, err
	}
	for _, p := range peersEnv.Peers {
		snap.Peers = append(snap.Peers, PeerRow{PeerID: p.PeerID, DisplayName: p.DisplayName})
	}

	var st struct {
		ProtoMajor int `json:"proto_major"`
		ProtoMinor int `json:"proto_minor"`
	}
	if err := c.getJSON("/status", &st); err != nil {
		return Snapshot{EngineOK: false, Err: err.Error()}, err
	}
	snap.ProtoMajor = st.ProtoMajor
	snap.ProtoMinor = st.ProtoMinor

	var msgsEnv struct {
		Messages []struct {
			DisplayNameAtSend string    `json:"display_name_at_send"`
			Text              string    `json:"text"`
			TS                time.Time `json:"ts"`
		} `json:"messages"`
	}
	if err := c.getJSON("/messages", &msgsEnv); err != nil {
		return Snapshot{EngineOK: false, Err: err.Error()}, err
	}
	for _, m := range msgsEnv.Messages {
		snap.Messages = append(snap.Messages, MsgRow{
			DisplayName: m.DisplayNameAtSend,
			Text:        m.Text,
			TS:          m.TS,
		})
	}

	snap.EngineOK = true
	return snap, nil
}

func (c *Client) getJSON(path string, dst any) error {
	resp, err := c.client.Get(c.base + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tui: %s → %s", path, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}
