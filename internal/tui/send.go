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
	if fileID, force, isFetch, err := ParseFetchCommand(text); isFetch {
		if err != nil {
			return err
		}
		plan, err := c.BeginFetch(fileID, force)
		if err != nil {
			return err
		}
		if plan.Warning != "" {
			return &ErrLargeFileWarning{FileID: fileID, Msg: plan.Warning}
		}
		return nil
	}
	if fileID, isCancel, err := ParseCancelCommand(text); isCancel {
		if err != nil {
			return err
		}
		_, err := c.CancelFetch(fileID)
		return err
	}
	_, err := c.Send(text)
	return err
}

// ErrLargeFileWarning is returned when /fetch needs confirmation for >100 MiB (P054).
// It is not a hard block — /fetch! or force proceeds.
type ErrLargeFileWarning struct {
	FileID string
	Msg    string
}

func (e *ErrLargeFileWarning) Error() string {
	if e == nil {
		return ""
	}
	return e.Msg
}

// ParseFetchCommand recognizes `/fetch <file_id>`, `/fetch! <id>`, `/fetch <id> --yes` (P052/P054).
func ParseFetchCommand(line string) (fileID string, force bool, ok bool, err error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false, false, nil
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", false, false, nil
	}
	cmd := fields[0]
	switch {
	case strings.EqualFold(cmd, "/fetch!"):
		force = true
	case strings.EqualFold(cmd, "/fetch"):
		force = false
	default:
		return "", false, false, nil
	}
	if len(fields) < 2 {
		return "", force, true, fmt.Errorf("tui: /fetch needs a file_id")
	}
	fileID = fields[1]
	for _, f := range fields[2:] {
		if f == "--yes" || f == "-y" || strings.EqualFold(f, "yes") {
			force = true
		}
	}
	if strings.TrimSpace(fileID) == "" {
		return "", force, true, fmt.Errorf("tui: /fetch needs a file_id")
	}
	return fileID, force, true, nil
}

// ParseCancelCommand recognizes `/cancel <file_id>` (P053).
func ParseCancelCommand(line string) (fileID string, ok bool, err error) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "/cancel") {
		return "", false, nil
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, "/cancel"))
	if rest == "" {
		return "", true, fmt.Errorf("tui: /cancel needs a file_id")
	}
	return rest, true, nil
}
