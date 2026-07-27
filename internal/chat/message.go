// Package chat owns in-memory text messages, file announces, and LAN fan-out.
package chat

import (
	"encoding/json"
	"fmt"
	"time"
)

// Message is a feed item on the wire and in local stores (text or file announce).
type Message struct {
	Type              string    `json:"type"`
	MsgID             string    `json:"msg_id"`
	PeerID            string    `json:"peer_id"`
	DisplayNameAtSend string    `json:"display_name_at_send"`
	TS                time.Time `json:"ts"`
	Text              string    `json:"text,omitempty"`
	FileID            string    `json:"file_id,omitempty"`
	FileName          string    `json:"name,omitempty"`
	Size              int64     `json:"size,omitempty"`
	Mime              string    `json:"mime,omitempty"`
	Hash              string    `json:"hash,omitempty"`
	ThumbB64          string    `json:"thumb_b64,omitempty"`  // small JPEG preview on the wire (P056)
	ThumbPath         string    `json:"thumb_path,omitempty"` // local materialized path; not on wire
}

// EncodeMessage serializes a feed line (newline-delimited JSON).
func EncodeMessage(m Message) ([]byte, error) {
	if m.TS.IsZero() {
		m.TS = time.Now().UTC()
	} else {
		m.TS = m.TS.UTC()
	}
	switch m.Type {
	case "", TypeChat:
		m.Type = TypeChat
		m.FileID, m.FileName, m.Mime, m.Hash = "", "", "", ""
		m.ThumbB64, m.ThumbPath = "", ""
		m.Size = 0
	case TypeFileAnnounce:
		if err := ValidateFileAnnounce(FileAnnounce{
			Name: m.FileName, Size: m.Size, Mime: m.Mime, Hash: m.Hash,
		}); err != nil {
			return nil, err
		}
		if m.FileID == "" {
			return nil, fmt.Errorf("chat: file_id required")
		}
		m.Text = ""
		m.ThumbPath = "" // local only — receivers materialize from thumb_b64
	default:
		return nil, fmt.Errorf("chat: unexpected type %q", m.Type)
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// DecodeMessage parses one feed JSON object (optional trailing newline).
func DecodeMessage(raw []byte) (Message, error) {
	var m Message
	if err := json.Unmarshal(raw, &m); err != nil {
		return Message{}, err
	}
	switch m.Type {
	case "", TypeChat:
		m.Type = TypeChat
		if m.MsgID == "" || m.PeerID == "" {
			return Message{}, fmt.Errorf("chat: missing msg_id/peer_id")
		}
	case TypeFileAnnounce:
		if m.MsgID == "" || m.PeerID == "" {
			return Message{}, fmt.Errorf("chat: missing msg_id/peer_id")
		}
		if m.FileID == "" {
			return Message{}, fmt.Errorf("chat: missing file_id")
		}
		if err := ValidateFileAnnounce(FileAnnounce{
			Name: m.FileName, Size: m.Size, Mime: m.Mime, Hash: m.Hash,
		}); err != nil {
			return Message{}, err
		}
		m.Text = ""
	default:
		return Message{}, fmt.Errorf("chat: unexpected type %q", m.Type)
	}
	m.TS = m.TS.UTC()
	return m, nil
}
