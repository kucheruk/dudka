package files

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// MaterializeLocal copies an internal source blob to a human-facing inbox
// path. The destination appears only after a complete copy; cancellation
// removes the partial file.
func MaterializeLocal(
	ctx context.Context,
	sourcePath string,
	destPath string,
	report func(received int64),
) (written int64, err error) {
	source, err := os.Open(sourcePath)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return 0, err
	}
	partial := destPath + ".partial"
	target, err := os.OpenFile(partial, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	closed := false
	defer func() {
		if !closed {
			_ = target.Close()
		}
		if err != nil {
			_ = os.Remove(partial)
		}
	}()

	buf := make([]byte, DefaultChunkSize)
	for {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return written, ErrCancelled
		}
		n, readErr := source.Read(buf)
		if n > 0 {
			wrote, writeErr := target.Write(buf[:n])
			written += int64(wrote)
			if writeErr != nil {
				return written, writeErr
			}
			if wrote != n {
				return written, io.ErrShortWrite
			}
			if report != nil {
				report(written)
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return written, readErr
		}
	}
	if err := target.Close(); err != nil {
		return written, err
	}
	closed = true
	if err := os.Rename(partial, destPath); err != nil {
		if removeErr := os.Remove(destPath); removeErr != nil &&
			!errors.Is(removeErr, os.ErrNotExist) {
			return written, err
		}
		if retryErr := os.Rename(partial, destPath); retryErr != nil {
			return written, retryErr
		}
	}
	return written, nil
}
