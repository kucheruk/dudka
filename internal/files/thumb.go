package files

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder
)

// ThumbMaxEdge is the longest side of a generated preview (P056 / DUD-FILE-120).
const ThumbMaxEdge = 96

// MaxThumbBytes caps on-wire thumb_b64 payload after JPEG encode.
const MaxThumbBytes = 48 << 10

// IsThumbMIME reports jpeg/png/webp/heic/heif candidates (P056/P057).
func IsThumbMIME(mime string) bool {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/jpeg", "image/jpg", "image/png", "image/webp":
		return true
	default:
		return IsHEICMIME(mime)
	}
}

// MakeThumb builds a small JPEG preview for supported image bytes.
// ok=false means "no thumb" (non-image / unsupported / HEIC without decoder) without failing announce.
func MakeThumb(data []byte, mime string) (thumbJPEG []byte, ok bool, err error) {
	if !IsThumbMIME(mime) || len(data) == 0 {
		return nil, false, nil
	}
	var src image.Image
	if IsHEICMIME(mime) {
		if !HEICAvailable() {
			return nil, false, nil // honest fallback — no fake preview
		}
		img, derr := decodeHEIC(data)
		if derr != nil {
			return nil, false, nil // corrupt / undecodable HEIC → no thumb, no error on announce
		}
		src = img
	} else {
		img, _, derr := image.Decode(bytes.NewReader(data))
		if derr != nil {
			return nil, false, fmt.Errorf("files: decode image: %w", derr)
		}
		src = img
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, false, fmt.Errorf("files: empty image")
	}
	nw, nh := w, h
	if w > ThumbMaxEdge || h > ThumbMaxEdge {
		if w >= h {
			nw = ThumbMaxEdge
			nh = h * ThumbMaxEdge / w
		} else {
			nh = ThumbMaxEdge
			nw = w * ThumbMaxEdge / h
		}
		if nw < 1 {
			nw = 1
		}
		if nh < 1 {
			nh = 1
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: 70}); err != nil {
		return nil, false, err
	}
	thumb := out.Bytes()
	if len(thumb) == 0 || len(thumb) > MaxThumbBytes {
		return nil, false, nil
	}
	return thumb, true, nil
}

// ThumbPath returns thumbsDir/<file_id>.jpg
func ThumbPath(thumbsDir, fileID string) (string, error) {
	thumbsDir = strings.TrimSpace(thumbsDir)
	if thumbsDir == "" {
		return "", fmt.Errorf("files: empty thumbs dir")
	}
	id := strings.TrimSpace(fileID)
	if id == "" || strings.ContainsAny(id, `/\`) || id == "." || id == ".." {
		return "", fmt.Errorf("files: bad file_id")
	}
	if err := os.MkdirAll(thumbsDir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(thumbsDir, id+".jpg"), nil
}

// WriteThumb writes JPEG thumb bytes to ThumbPath.
func WriteThumb(thumbsDir, fileID string, thumbJPEG []byte) (string, error) {
	if len(thumbJPEG) == 0 {
		return "", fmt.Errorf("files: empty thumb")
	}
	p, err := ThumbPath(thumbsDir, fileID)
	if err != nil {
		return "", err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, thumbJPEG, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return p, nil
}
