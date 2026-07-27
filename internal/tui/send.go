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

// HandleComposeLine handles compose input: /nick, /fetch, or Enter = send.
// Blank lines are ignored (no error).
func HandleComposeLine(c *Client, line string) error {
	text := strings.TrimSpace(line)
	if text == "" {
		return nil
	}
	if name, isNick, err := ParseNickCommand(text); isNick {
		if err != nil {
			return err
		}
		_, err := c.SetNick(name)
		return err
	}
	if fileID, isFetch, err := ParseFetchCommand(text); isFetch {
		if err != nil {
			return err
		}
		_, err := c.StartFetch(fileID)
		return err
	}
	_, err := c.Send(text)
	return err
}

// ParseFetchCommand recognizes `/fetch <file_id>` (P052).
func ParseFetchCommand(line string) (fileID string, ok bool, err error) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "/fetch") {
		return "", false, nil
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, "/fetch"))
	if rest == "" {
		return "", true, fmt.Errorf("tui: /fetch needs a file_id")
	}
	return rest, true, nil
}
