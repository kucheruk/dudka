package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// SendResult is a subset of engine POST /send response (P035 statuses).
type SendResult struct {
	Status string `json:"status"`
	Queued int    `json:"queued"`
}

// Send posts text to engine POST /send.
func (c *Client) Send(text string) (SendResult, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return SendResult{}, fmt.Errorf("tui: text is required")
	}
	if c.base == "" {
		return SendResult{}, fmt.Errorf("tui: engine URL empty")
	}
	body, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return SendResult{}, err
	}
	resp, err := c.client.Post(c.base+"/send", "application/json", bytes.NewReader(body))
	if err != nil {
		return SendResult{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = resp.Status
		}
		return SendResult{}, fmt.Errorf("tui: send → %s", msg)
	}
	var res SendResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return SendResult{}, fmt.Errorf("tui: send decode: %w", err)
	}
	return res, nil
}

// HandleComposeLine trims a compose line and sends non-empty text (Enter = send).
// Blank lines are ignored (no error).
func HandleComposeLine(c *Client, line string) error {
	text := strings.TrimSpace(line)
	if text == "" {
		return nil
	}
	_, err := c.Send(text)
	return err
}
