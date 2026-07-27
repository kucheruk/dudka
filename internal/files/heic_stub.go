//go:build !(darwin && cgo)

package files

import (
	"fmt"
	"image"
)

// HEICAvailable is false when ImageIO/CGO HEIC decode is not in this build (P057).
func HEICAvailable() bool { return false }

func decodeHEIC(_ []byte) (image.Image, error) {
	return nil, fmt.Errorf("files: heic decode unavailable on this platform")
}
