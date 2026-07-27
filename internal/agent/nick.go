// Package agent validates home-agent display names (DUD-AGT-110).
package agent

import (
	"fmt"
	"strings"
	"unicode"
)

const sep = "·" // U+00B7 middle dot

// ValidateAgentNick checks `{agent}·{model}·{host}` (exactly two separators, three non-empty segments).
func ValidateAgentNick(name string) error {
	if name != strings.TrimSpace(name) {
		return fmt.Errorf("agent: leading/trailing space")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("agent: empty nick")
	}
	parts := strings.Split(name, sep)
	if len(parts) != 3 {
		return fmt.Errorf("agent: want exactly 3 segments separated by %q, got %d", sep, len(parts))
	}
	for i, p := range parts {
		if strings.TrimSpace(p) == "" || p != strings.TrimSpace(p) {
			return fmt.Errorf("agent: segment %d empty or padded", i)
		}
		for _, r := range p {
			if r == '·' || unicode.IsSpace(r) {
				return fmt.Errorf("agent: segment %d has space or extra separator", i)
			}
		}
	}
	return nil
}

// LooksLikeAgentNick reports whether name has the triple-prefix shape (humans must not use this).
func LooksLikeAgentNick(name string) bool {
	return ValidateAgentNick(name) == nil
}

// FormatAgentNick builds a canonical agent nick.
func FormatAgentNick(agentName, modelID, hostID string) (string, error) {
	out := strings.Join([]string{agentName, modelID, hostID}, sep)
	if err := ValidateAgentNick(out); err != nil {
		return "", err
	}
	return out, nil
}
