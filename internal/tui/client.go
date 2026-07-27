package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
		ProtoMajor int    `json:"proto_major"`
		ProtoMinor int    `json:"proto_minor"`
		Network    string `json:"network"`
	}
	if err := c.getJSON("/status", &st); err != nil {
		return Snapshot{EngineOK: false, Err: err.Error()}, err
	}
	snap.ProtoMajor = st.ProtoMajor
	snap.ProtoMinor = st.ProtoMinor
	snap.Network = st.Network
	if snap.Network == "" {
		snap.Network = NetworkOK
	}

	var msgsEnv struct {
		Messages []struct {
			Type              string    `json:"type"`
			DisplayNameAtSend string    `json:"display_name_at_send"`
			Text              string    `json:"text"`
			TS                time.Time `json:"ts"`
			FileID            string    `json:"file_id"`
			FileName          string    `json:"name"`
			Size              int64     `json:"size"`
			Mime              string    `json:"mime"`
			Hash              string    `json:"hash"`
		} `json:"messages"`
	}
	if err := c.getJSON("/messages", &msgsEnv); err != nil {
		return Snapshot{EngineOK: false, Err: err.Error()}, err
	}
	for _, m := range msgsEnv.Messages {
		typ := m.Type
		if typ == "" {
			typ = MsgTypeChat
		}
		snap.Messages = append(snap.Messages, MsgRow{
			DisplayName: m.DisplayNameAtSend,
			Text:        m.Text,
			TS:          m.TS,
			Type:        typ,
			FileID:      m.FileID,
			FileName:    m.FileName,
			Size:        m.Size,
			Mime:        m.Mime,
			Hash:        m.Hash,
		})
	}

	var xferEnv struct {
		Transfers []struct {
			FileID  string `json:"file_id"`
			Name    string `json:"name"`
			Percent int    `json:"percent"`
			Status  string `json:"status"`
		} `json:"transfers"`
	}
	if err := c.getJSON("/files/transfers", &xferEnv); err != nil {
		return Snapshot{EngineOK: false, Err: err.Error()}, err
	}
	for _, tr := range xferEnv.Transfers {
		snap.Transfers = append(snap.Transfers, TransferRow{
			FileID:  tr.FileID,
			Name:    tr.Name,
			Percent: tr.Percent,
			Status:  tr.Status,
		})
	}

	snap.EngineOK = true
	return snap, nil
}

// StartFetch begins an async download; progress appears on GET /files/transfers (P052).
func (c *Client) StartFetch(fileID string) (TransferRow, error) {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return TransferRow{}, fmt.Errorf("tui: empty file_id")
	}
	wait := false
	body, _ := json.Marshal(map[string]any{"file_id": fileID, "wait": wait})
	resp, err := c.client.Post(c.base+"/files/fetch", "application/json", bytes.NewReader(body))
	if err != nil {
		return TransferRow{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return TransferRow{}, fmt.Errorf("tui: fetch → %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
	var tr TransferRow
	var wire struct {
		FileID  string `json:"file_id"`
		Name    string `json:"name"`
		Percent int    `json:"percent"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return TransferRow{}, err
	}
	tr.FileID = wire.FileID
	tr.Name = wire.Name
	tr.Percent = wire.Percent
	tr.Status = wire.Status
	return tr, nil
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
