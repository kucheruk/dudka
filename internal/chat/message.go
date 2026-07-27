// Package chat owns in-memory text messages and LAN fan-out (DUD-CHAT-101).
package chat

import (
	"encoding/json"
	"fmt"
	"time"
)

// Message is a chat text payload on the wire and in local stores.
type Message struct {
	Type              string    `json:"type"`
	MsgID             string    `json:"msg_id"`
	PeerID            string    `json:"peer_id"`
	DisplayNameAtSend string    `json:"display_name_at_send"`
	TS                time.Time `json:"ts"`
	Text              string    `json:"text"`
}

// EncodeMessage serializes a chat line (newline-delimited JSON).
func EncodeMessage(m Message) ([]byte, error) {
	m.Type = "chat"
	if m.TS.IsZero() {
		m.TS = time.Now().UTC()
	} else {
		m.TS = m.TS.UTC()
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// DecodeMessage parses one chat JSON object (optional trailing newline).
func DecodeMessage(raw []byte) (Message, error) {
	var m Message
	if err := json.Unmarshal(raw, &m); err != nil {
		return Message{}, err
	}
	if m.Type != "" && m.Type != "chat" {
		return Message{}, fmt.Errorf("chat: unexpected type %q", m.Type)
	}
	m.Type = "chat"
	if m.MsgID == "" || m.PeerID == "" {
		return Message{}, fmt.Errorf("chat: missing msg_id/peer_id")
	}
	m.TS = m.TS.UTC()
	return m, nil
}
