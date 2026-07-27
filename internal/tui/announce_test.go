package tui_test

import (
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dudka/internal/files"
	"dudka/internal/tui"
)

func TestDetectMIME(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"a.jpg":  "image/jpeg",
		"a.JPEG": "image/jpeg",
		"a.png":  "image/png",
		"a.webp": "image/webp",
		"a.heic": "image/heic",
		"a.bin":  "application/octet-stream",
		"notes":  "application/octet-stream",
	}
	for name, want := range cases {
		if got := tui.DetectMIME(name); got != want {
			t.Fatalf("%s: got %q want %q", name, got, want)
		}
	}
}

func TestAnnouncePathPostsFileAndReturnsThumbForJPEG(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "pic.jpg")
	img := image.NewRGBA(image.Rect(0, 0, 24, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 24; x++ {
			img.Set(x, y, color.RGBA{R: 10, G: 120, B: 200, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 85}); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := files.SHA256Sum(raw)

	var gotBody map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("POST /files/announce", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode: %v", err)
			http.Error(w, "bad", 400)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "accepted",
			"message": map[string]any{
				"type":       "file_announce",
				"file_id":    "fid-img",
				"name":       gotBody["name"],
				"mime":       gotBody["mime"],
				"hash":       gotBody["hash"],
				"size":       gotBody["size"],
				"thumb_path": "/tmp/thumbs/fid-img.jpg",
				"thumb_b64":  base64.StdEncoding.EncodeToString([]byte{0xff, 0xd8, 0xff}),
			},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	res, err := tui.NewClient(srv.URL).AnnouncePath(path)
	if err != nil {
		t.Fatal(err)
	}
	if res.FileID != "fid-img" || res.Name != "pic.jpg" || res.Mime != "image/jpeg" {
		t.Fatalf("res=%+v", res)
	}
	if res.Hash != wantHash {
		t.Fatalf("hash=%q want %q", res.Hash, wantHash)
	}
	if gotBody["mime"] != "image/jpeg" || gotBody["hash"] != wantHash {
		t.Fatalf("posted=%v", gotBody)
	}
	b64, _ := gotBody["content_b64"].(string)
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || string(decoded) != string(raw) {
		t.Fatalf("content mismatch err=%v", err)
	}
}

func TestAnnouncePathBinaryNoThumbRequired(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "payload.bin")
	payload := []byte{0x00, 0x01, 0x02, 0xde, 0xad}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /files/announce", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["mime"] != "application/octet-stream" {
			t.Errorf("mime=%v", body["mime"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "accepted",
			"message": map[string]any{
				"type": "file_announce", "file_id": "fid-bin",
				"name": "payload.bin", "mime": "application/octet-stream",
				"hash": body["hash"], "size": body["size"],
			},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	res, err := tui.NewClient(srv.URL).AnnouncePath(path)
	if err != nil {
		t.Fatal(err)
	}
	if res.FileID != "fid-bin" || res.ThumbPath != "" {
		t.Fatalf("res=%+v", res)
	}
	if !strings.Contains(res.Mime, "octet-stream") {
		t.Fatalf("mime=%q", res.Mime)
	}
}
