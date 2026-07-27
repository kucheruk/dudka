package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// SetNick posts a new display name to engine POST /nick (P043).
func (c *Client) SetNick(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("tui: nick is required")
	}
	if c.base == "" {
		return "", fmt.Errorf("tui: engine URL empty")
	}
	body, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		return "", err
	}
	resp, err := c.client.Post(c.base+"/nick", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = resp.Status
		}
		return "", fmt.Errorf("tui: nick → %s", msg)
	}
	var me struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &me); err != nil {
		return "", fmt.Errorf("tui: nick decode: %w", err)
	}
	if me.Name == "" {
		me.Name = name
	}
	return me.Name, nil
}

// ParseNickCommand extracts a nick from "/nick Name" (case-insensitive command).
// ok=false when line is not a nick command.
func ParseNickCommand(line string) (name string, ok bool, err error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false, nil
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", false, nil
	}
	if !strings.EqualFold(fields[0], "/nick") {
		return "", false, nil
	}
	if len(fields) < 2 {
		return "", true, fmt.Errorf("tui: usage: /nick Имя")
	}
	name = strings.TrimSpace(strings.Join(fields[1:], " "))
	if name == "" {
		return "", true, fmt.Errorf("tui: usage: /nick Имя")
	}
	return name, true, nil
}
