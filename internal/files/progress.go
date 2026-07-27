package files

import "errors"

// ErrCancelled is returned when a chunk download is aborted (P053).
var ErrCancelled = errors.New("files: download cancelled")

// Percent maps received/total bytes to 0–100 (DUD-FILE-110 / P052).
func Percent(received, total int64) int {
	if total <= 0 {
		if received > 0 {
			return 100
		}
		return 0
	}
	if received <= 0 {
		return 0
	}
	if received >= total {
		return 100
	}
	p := int(received * 100 / total)
	if p < 1 {
		// Any positive progress should be visible as at least 1%.
		return 1
	}
	if p > 100 {
		return 100
	}
	return p
}

// ProgressFunc is invoked as chunks are written (received may be < total until EOF).
type ProgressFunc func(received, total int64)
