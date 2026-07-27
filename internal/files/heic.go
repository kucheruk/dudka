package files

import "strings"

// IsHEICMIME reports image/heic or image/heif (P057 / DUD-FILE-120).
func IsHEICMIME(mime string) bool {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/heic", "image/heif":
		return true
	default:
		return false
	}
}
