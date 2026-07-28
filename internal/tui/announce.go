package tui

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dudka/internal/files"
)

// AnnounceResult is the loopback outcome of posting a local file (P058).
type AnnounceResult struct {
	FileID    string
	Name      string
	Mime      string
	Hash      string
	Size      int64
	ThumbPath string
}

// DetectMIME maps a file name to announce mime (extension-based; unknown → octet-stream).
func DetectMIME(name string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(name)))
	switch ext {
	case ".gif":
		return "image/gif"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".heic", ".heif":
		return "image/heic"
	default:
		return "application/octet-stream"
	}
}

// AnnouncePath reads a local file and publishes file_announce via the engine (P058).
func (c *Client) AnnouncePath(path string) (AnnounceResult, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return AnnounceResult{}, fmt.Errorf("tui: empty announce path")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return AnnounceResult{}, err
	}
	name := filepath.Base(path)
	mime := DetectMIME(name)
	hash := files.SHA256Sum(raw)
	body, err := json.Marshal(map[string]any{
		"name":        name,
		"mime":        mime,
		"hash":        hash,
		"size":        len(raw),
		"content_b64": base64.StdEncoding.EncodeToString(raw),
	})
	if err != nil {
		return AnnounceResult{}, err
	}
	// Announce may carry content_b64; allow a longer round-trip than snapshot polls.
	client := c.client
	if client.Timeout < 15*time.Second {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := client.Post(c.base+"/files/announce", "application/json", bytes.NewReader(body))
	if err != nil {
		return AnnounceResult{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return AnnounceResult{}, err
	}
	if resp.StatusCode >= 300 {
		return AnnounceResult{}, fmt.Errorf("tui: announce: %s", strings.TrimSpace(string(data)))
	}
	var env struct {
		Message struct {
			FileID    string `json:"file_id"`
			Name      string `json:"name"`
			Mime      string `json:"mime"`
			Hash      string `json:"hash"`
			Size      int64  `json:"size"`
			ThumbPath string `json:"thumb_path"`
		} `json:"message"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return AnnounceResult{}, err
	}
	if env.Message.FileID == "" {
		return AnnounceResult{}, fmt.Errorf("tui: announce missing file_id")
	}
	return AnnounceResult{
		FileID:    env.Message.FileID,
		Name:      env.Message.Name,
		Mime:      env.Message.Mime,
		Hash:      env.Message.Hash,
		Size:      env.Message.Size,
		ThumbPath: env.Message.ThumbPath,
	}, nil
}
