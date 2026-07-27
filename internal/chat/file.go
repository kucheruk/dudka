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
// Optional Content is retained locally for chunk serving (P051); it is never fan-out on announce.
type FileAnnounce struct {
	Name    string
	Size    int64
	Mime    string
	Hash    string
	Content []byte // optional local blob for the source peer
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
