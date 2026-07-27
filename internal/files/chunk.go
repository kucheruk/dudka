package files

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DefaultChunkSize is the LAN transfer chunk size when limit is unset.
const DefaultChunkSize int64 = 64 * 1024

// Wire types for chunked file transfer.
const (
	TypeChunkReq = "file_chunk_req"
	TypeChunk    = "file_chunk"
)

// ChunkReq asks a source peer for file bytes starting at offset.
type ChunkReq struct {
	Type   string `json:"type"`
	FileID string `json:"file_id"`
	Offset int64  `json:"offset"`
	Limit  int64  `json:"limit"`
}

// Chunk is one base64 payload slice on the wire.
type Chunk struct {
	Type   string `json:"type"`
	FileID string `json:"file_id"`
	Offset int64  `json:"offset"`
	Data   string `json:"data"`
	EOF    bool   `json:"eof"`
}

// EncodeChunkReq serializes a chunk request line.
func EncodeChunkReq(req ChunkReq) ([]byte, error) {
	req.Type = TypeChunkReq
	req.FileID = strings.TrimSpace(req.FileID)
	if req.FileID == "" {
		return nil, fmt.Errorf("files: file_id required")
	}
	if req.Offset < 0 {
		return nil, fmt.Errorf("files: offset must be >= 0")
	}
	if req.Limit < 0 {
		return nil, fmt.Errorf("files: limit must be >= 0")
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// DecodeChunkReq parses a chunk request line.
func DecodeChunkReq(raw []byte) (ChunkReq, error) {
	var req ChunkReq
	if err := json.Unmarshal(raw, &req); err != nil {
		return ChunkReq{}, err
	}
	if req.Type != "" && req.Type != TypeChunkReq {
		return ChunkReq{}, fmt.Errorf("files: unexpected type %q", req.Type)
	}
	req.Type = TypeChunkReq
	req.FileID = strings.TrimSpace(req.FileID)
	if req.FileID == "" {
		return ChunkReq{}, fmt.Errorf("files: file_id required")
	}
	if req.Offset < 0 || req.Limit < 0 {
		return ChunkReq{}, fmt.Errorf("files: bad offset/limit")
	}
	return req, nil
}

// EncodeChunk serializes one chunk response line.
func EncodeChunk(c Chunk) ([]byte, error) {
	c.Type = TypeChunk
	raw, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// DecodeChunk parses one chunk response line.
func DecodeChunk(raw []byte) (Chunk, error) {
	var c Chunk
	if err := json.Unmarshal(raw, &c); err != nil {
		return Chunk{}, err
	}
	if c.Type != "" && c.Type != TypeChunk {
		return Chunk{}, fmt.Errorf("files: unexpected type %q", c.Type)
	}
	c.Type = TypeChunk
	return c, nil
}

// ServeChunks writes chunk lines for req until EOF (inclusive empty eof chunk when empty file).
func ServeChunks(w io.Writer, store *Store, req ChunkReq) error {
	if store == nil {
		return fmt.Errorf("files: nil store")
	}
	req.FileID = strings.TrimSpace(req.FileID)
	if req.FileID == "" {
		return fmt.Errorf("files: file_id required")
	}
	limit := req.Limit
	if limit <= 0 {
		limit = DefaultChunkSize
	}
	f, err := store.Open(req.FileID)
	if err != nil {
		return err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return err
	}
	size := st.Size()
	offset := req.Offset
	if offset > size {
		return fmt.Errorf("files: offset past end")
	}
	buf := make([]byte, limit)
	for {
		n, err := f.ReadAt(buf, offset)
		if n > 0 {
			raw, encErr := EncodeChunk(Chunk{
				FileID: req.FileID,
				Offset: offset,
				Data:   base64.StdEncoding.EncodeToString(buf[:n]),
				EOF:    offset+int64(n) >= size,
			})
			if encErr != nil {
				return encErr
			}
			if _, werr := w.Write(raw); werr != nil {
				return werr
			}
			offset += int64(n)
		}
		if err == io.EOF || offset >= size {
			if n == 0 && size == 0 {
				raw, encErr := EncodeChunk(Chunk{
					FileID: req.FileID,
					Offset: 0,
					Data:   "",
					EOF:    true,
				})
				if encErr != nil {
					return encErr
				}
				_, werr := w.Write(raw)
				return werr
			}
			if n == 0 && offset < size {
				return fmt.Errorf("files: short read")
			}
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// ReadChunks consumes chunk lines from r and writes the reassembled file to destPath.
// total is the expected size from file-announce (for percent); onProgress may be nil.
// ctx cancel aborts the download, discards the partial file, and returns ErrCancelled (P053).
func ReadChunks(ctx context.Context, r io.Reader, fileID string, destPath string, total int64, onProgress ProgressFunc) (int64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	br := bufio.NewReader(r)
	tmp := destPath + ".partial"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	closed := false
	defer func() {
		if !closed {
			_ = f.Close()
		}
		if tmp != "" {
			_ = os.Remove(tmp)
			_ = os.Remove(destPath) // never leave a success path after abort
		}
	}()

	report := func(written int64) {
		if onProgress != nil {
			onProgress(written, total)
		}
	}
	report(0)

	var written int64
	expect := int64(0)
	for {
		if err := ctx.Err(); err != nil {
			return written, ErrCancelled
		}
		line, err := br.ReadBytes('\n')
		if err != nil && !(err == io.EOF && len(line) > 0) {
			if err == io.EOF {
				return written, fmt.Errorf("files: unexpected EOF before chunk eof")
			}
			return written, err
		}
		c, decErr := DecodeChunk(line)
		if decErr != nil {
			return written, decErr
		}
		if c.FileID != "" && c.FileID != fileID {
			return written, fmt.Errorf("files: file_id mismatch")
		}
		if c.Offset != expect {
			return written, fmt.Errorf("files: unexpected offset %d want %d", c.Offset, expect)
		}
		var data []byte
		if c.Data != "" {
			data, decErr = base64.StdEncoding.DecodeString(c.Data)
			if decErr != nil {
				return written, decErr
			}
		}
		if len(data) > 0 {
			if _, werr := f.Write(data); werr != nil {
				return written, werr
			}
			written += int64(len(data))
			expect += int64(len(data))
			report(written)
			if err := ctx.Err(); err != nil {
				return written, ErrCancelled
			}
		}
		if c.EOF {
			if err := ctx.Err(); err != nil {
				return written, ErrCancelled
			}
			if cerr := f.Close(); cerr != nil {
				return written, cerr
			}
			closed = true
			if rerr := os.Rename(tmp, destPath); rerr != nil {
				return written, rerr
			}
			tmp = ""
			if total > 0 {
				report(total)
			} else {
				report(written)
			}
			return written, nil
		}
		if err == io.EOF {
			return written, fmt.Errorf("files: stream ended without eof")
		}
	}
}

// InboxPath builds a safe destination path under inboxDir.
func InboxPath(inboxDir, fileID, name string) (string, error) {
	inboxDir = strings.TrimSpace(inboxDir)
	if inboxDir == "" {
		return "", fmt.Errorf("files: empty inbox dir")
	}
	id := strings.TrimSpace(fileID)
	if id == "" || strings.ContainsAny(id, `/\`) {
		return "", fmt.Errorf("files: bad file_id")
	}
	base := filepath.Base(strings.TrimSpace(name))
	if base == "." || base == string(filepath.Separator) || base == "" {
		base = "file"
	}
	if err := os.MkdirAll(inboxDir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(inboxDir, id+"_"+base), nil
}
