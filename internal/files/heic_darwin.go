//go:build darwin && cgo

package files

/*
#cgo LDFLAGS: -framework ImageIO -framework CoreFoundation -framework CoreGraphics
#include <CoreFoundation/CoreFoundation.h>
#include <ImageIO/ImageIO.h>
#include <stdlib.h>
#include <string.h>

// decode_heic_rgba decodes the first frame into a newly malloc'd RGBA buffer.
// Caller must free(*out) with free(). Returns 0 on success.
int decode_heic_rgba(const uint8_t *data, size_t n, uint8_t **out, int *w, int *h) {
  if (!data || n == 0 || !out || !w || !h) return -1;
  *out = NULL; *w = 0; *h = 0;
  CFDataRef cf = CFDataCreate(NULL, data, (CFIndex)n);
  if (!cf) return -2;
  CGImageSourceRef src = CGImageSourceCreateWithData(cf, NULL);
  CFRelease(cf);
  if (!src) return -3;
  CGImageRef img = CGImageSourceCreateImageAtIndex(src, 0, NULL);
  CFRelease(src);
  if (!img) return -4;
  size_t width = CGImageGetWidth(img);
  size_t height = CGImageGetHeight(img);
  if (width == 0 || height == 0) {
    CGImageRelease(img);
    return -5;
  }
  size_t bytes = width * height * 4;
  uint8_t *buf = (uint8_t *)malloc(bytes);
  if (!buf) {
    CGImageRelease(img);
    return -6;
  }
  memset(buf, 0, bytes);
  CGColorSpaceRef cs = CGColorSpaceCreateDeviceRGB();
  CGContextRef ctx = CGBitmapContextCreate(
      buf, width, height, 8, width * 4, cs,
      kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big);
  CGColorSpaceRelease(cs);
  if (!ctx) {
    free(buf);
    CGImageRelease(img);
    return -7;
  }
  CGContextDrawImage(ctx, CGRectMake(0, 0, (CGFloat)width, (CGFloat)height), img);
  CGContextRelease(ctx);
  CGImageRelease(img);
  *out = buf;
  *w = (int)width;
  *h = (int)height;
  return 0;
}
*/
import "C"

import (
	"fmt"
	"image"
	"image/color"
	"unsafe"
)

// HEICAvailable reports whether this build can decode HEIC/HEIF.
func HEICAvailable() bool { return true }

func decodeHEIC(data []byte) (image.Image, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("files: empty heic")
	}
	var out *C.uint8_t
	var w, h C.int
	rc := C.decode_heic_rgba((*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)), &out, &w, &h)
	if rc != 0 || out == nil || w <= 0 || h <= 0 {
		return nil, fmt.Errorf("files: heic decode failed (%d)", int(rc))
	}
	defer C.free(unsafe.Pointer(out))

	width, height := int(w), int(h)
	n := width * height * 4
	src := C.GoBytes(unsafe.Pointer(out), C.int(n))
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// ImageIO wrote premultiplied RGBA; unpremultiply lightly for thumb quality.
	for i := 0; i < width*height; i++ {
		o := i * 4
		a := src[o+3]
		r, g, b := src[o], src[o+1], src[o+2]
		if a > 0 && a < 255 {
			r = uint8(int(r) * 255 / int(a))
			g = uint8(int(g) * 255 / int(a))
			b = uint8(int(b) * 255 / int(a))
		}
		img.SetRGBA(i%width, i/width, color.RGBA{R: r, G: g, B: b, A: a})
	}
	return img, nil
}
