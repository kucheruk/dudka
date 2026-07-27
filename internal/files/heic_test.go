package files_test

import (
	"bytes"
	"image/jpeg"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"dudka/internal/files"
)

func TestIsHEICMIME(t *testing.T) {
	t.Parallel()
	for _, mime := range []string{"image/heic", "image/heif", "IMAGE/HEIC"} {
		if !files.IsHEICMIME(mime) {
			t.Fatalf("want HEIC mime %q", mime)
		}
	}
	if files.IsHEICMIME("image/jpeg") {
		t.Fatal("jpeg is not HEIC")
	}
}

func TestMakeThumbHEICGarbageHonestFallback(t *testing.T) {
	t.Parallel()
	thumb, ok, err := files.MakeThumb([]byte("not-really-heic"), "image/heic")
	if err != nil {
		t.Fatal(err)
	}
	if ok || thumb != nil {
		t.Fatalf("garbage HEIC must not invent thumb: ok=%v len=%d", ok, len(thumb))
	}
}

func TestMakeThumbHEICSample(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "testdata", "sample.heic")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	thumb, ok, err := files.MakeThumb(data, "image/heic")
	if err != nil {
		t.Fatal(err)
	}
	if !files.HEICAvailable() {
		if ok || thumb != nil {
			t.Fatalf("platform without HEIC decode must honest-fallback: ok=%v", ok)
		}
		t.Logf("HEIC unavailable on %s/%s — fallback OK", runtime.GOOS, runtime.GOARCH)
		return
	}
	if !ok || len(thumb) == 0 {
		t.Fatalf("want HEIC thumb on this platform, ok=%v len=%d", ok, len(thumb))
	}
	if !bytes.HasPrefix(thumb, []byte{0xff, 0xd8}) {
		t.Fatalf("thumb must be JPEG, got %x", thumb[:min(4, len(thumb))])
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(thumb))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width > files.ThumbMaxEdge || cfg.Height > files.ThumbMaxEdge {
		t.Fatalf("thumb too large: %dx%d", cfg.Width, cfg.Height)
	}
}

func TestHEICAvailableMatchesBuild(t *testing.T) {
	t.Parallel()
	avail := files.HEICAvailable()
	// Documented contract: darwin+cgo may decode; others must report false.
	if runtime.GOOS != "darwin" && avail {
		t.Fatalf("non-darwin must not claim HEICAvailable")
	}
}
