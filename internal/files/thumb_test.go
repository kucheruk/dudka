package files_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"dudka/internal/files"
)

func TestMakeThumbJPEGAndPNG(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mime string
		enc  func(*bytes.Buffer, image.Image) error
	}{
		{"image/jpeg", func(b *bytes.Buffer, img image.Image) error {
			return jpeg.Encode(b, img, &jpeg.Options{Quality: 90})
		}},
		{"image/png", func(b *bytes.Buffer, img image.Image) error {
			return png.Encode(b, img)
		}},
	} {
		tc := tc
		t.Run(tc.mime, func(t *testing.T) {
			t.Parallel()
			src := image.NewRGBA(image.Rect(0, 0, 64, 48))
			for y := 0; y < 48; y++ {
				for x := 0; x < 64; x++ {
					src.Set(x, y, color.RGBA{R: 200, G: 40, B: 40, A: 255})
				}
			}
			var buf bytes.Buffer
			if err := tc.enc(&buf, src); err != nil {
				t.Fatal(err)
			}
			thumb, ok, err := files.MakeThumb(buf.Bytes(), tc.mime)
			if err != nil {
				t.Fatal(err)
			}
			if !ok || len(thumb) == 0 {
				t.Fatalf("ok=%v len=%d", ok, len(thumb))
			}
			if !bytes.HasPrefix(thumb, []byte{0xff, 0xd8}) {
				t.Fatalf("thumb must be JPEG, got prefix %x", thumb[:min(4, len(thumb))])
			}
			cfg, err := jpeg.DecodeConfig(bytes.NewReader(thumb))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Width > files.ThumbMaxEdge || cfg.Height > files.ThumbMaxEdge {
				t.Fatalf("thumb too large: %dx%d", cfg.Width, cfg.Height)
			}
		})
	}
}

func TestMakeThumbWebP(t *testing.T) {
	t.Parallel()
	// 16×16 red WebP (PIL-generated fixture).
	webp := []byte{
		82, 73, 70, 70, 60, 0, 0, 0, 87, 69, 66, 80, 86, 80, 56, 32, 48, 0, 0, 0, 208, 1, 0, 157, 1, 42, 16, 0, 16, 0, 1, 64, 38, 37, 160, 2, 116, 186, 1, 248, 0, 3, 176, 0, 254, 242, 235, 127, 252, 216, 21, 205, 115, 239, 247, 255, 210, 224, 253, 46, 15, 210, 224, 255, 210, 144, 0, 0,
	}
	thumb, ok, err := files.MakeThumb(webp, "image/webp")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(thumb) == 0 {
		t.Fatalf("ok=%v len=%d", ok, len(thumb))
	}
}

func TestMakeThumbNonImageNoFake(t *testing.T) {
	t.Parallel()
	thumb, ok, err := files.MakeThumb([]byte("not-an-image"), "application/octet-stream")
	if err != nil {
		t.Fatal(err)
	}
	if ok || thumb != nil {
		t.Fatalf("non-image must not invent thumb: ok=%v len=%d", ok, len(thumb))
	}
}

func TestIsThumbMIME(t *testing.T) {
	t.Parallel()
	for _, mime := range []string{"image/jpeg", "image/png", "image/webp", "IMAGE/JPEG", "image/heic", "image/heif"} {
		if !files.IsThumbMIME(mime) {
			t.Fatalf("want thumb mime %q", mime)
		}
	}
	for _, mime := range []string{"text/plain", "application/octet-stream"} {
		if files.IsThumbMIME(mime) {
			t.Fatalf("must not be thumb mime %q", mime)
		}
	}
}
