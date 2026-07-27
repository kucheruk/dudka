package chat

import (
	"fmt"
	"strings"
)

// Wire / feed message types.
const (
	TypeChat         = "chat"
	TypeFileAnnounce = "file_announce"
)

// FileAnnounce is the metadata published into the chat feed (DUD-FILE-101 / P050).
// It does not carry file bytes — download is a later step (P051).
type FileAnnounce struct {
	Name string
	Size int64
	Mime string
	Hash string
}

// ValidateFileAnnounce checks required announce fields.
func ValidateFileAnnounce(a FileAnnounce) error {
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("chat: file name required")
	}
	if a.Size < 0 {
		return fmt.Errorf("chat: file size must be >= 0")
	}
	if strings.TrimSpace(a.Mime) == "" {
		return fmt.Errorf("chat: file mime required")
	}
	if strings.TrimSpace(a.Hash) == "" {
		return fmt.Errorf("chat: file hash required")
	}
	return nil
}
