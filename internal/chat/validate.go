package chat

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

// MaxTextCodePoints is the DUD-CHAT-101 / P031 limit for message text.
const MaxTextCodePoints = 4000

// ErrTextTooLong is returned when text exceeds MaxTextCodePoints UTF-8 code points.
var ErrTextTooLong = errors.New("chat: text exceeds 4000 characters")

// ValidateText checks non-empty text against the code-point limit.
func ValidateText(text string) error {
	if text == "" {
		return fmt.Errorf("chat: text is required")
	}
	if utf8.RuneCountInString(text) > MaxTextCodePoints {
		return fmt.Errorf("%w (got %d)", ErrTextTooLong, utf8.RuneCountInString(text))
	}
	return nil
}
