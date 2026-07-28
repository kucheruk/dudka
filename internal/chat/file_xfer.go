package chat

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"dudka/internal/discovery"
	"dudka/internal/files"
)

// HandleFileChunkRequest serves local blob chunks on an accepted TCP session (P051).
func (h *Hub) HandleFileChunkRequest(_ string, conn net.Conn, line []byte) {
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	req, err := files.DecodeChunkReq(line)
	if err != nil {
		h.logf("file_chunk_req_err err=%v", err)
		return
	}
	if h.blobs == nil || !h.blobs.Has(req.FileID) {
		h.logf("file_chunk_missing file_id=%s", req.FileID)
		return
	}
	if req.Limit <= 0 {
		req.Limit = h.chunkSize
	}
	if err := files.ServeChunks(conn, h.blobs, req); err != nil {
		h.logf("file_chunk_serve_err file_id=%s err=%v", req.FileID, err)
		return
	}
	h.logf("file_chunk_serve_ok file_id=%s", req.FileID)
}

// FetchResult is the loopback outcome of a completed download (P051).
type FetchResult struct {
	FileID  string `json:"file_id"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Name    string `json:"name"`
	Percent int    `json:"percent"`
	Status  string `json:"status"`
}

// FetchFile downloads file_id from its announce source peer via chunks and writes inbox path.
func (h *Hub) FetchFile(fileID string) (string, error) {
	res, err := h.Fetch(fileID)
	if err != nil {
		return "", err
	}
	return res.Path, nil
}

// StartFetch begins a download in the background (P052); progress via Transfers().
func (h *Hub) StartFetch(fileID string) (Transfer, error) {
	msg, ok := h.store.FindFile(fileID)
	if !ok {
		return Transfer{}, fmt.Errorf("chat: unknown file_id")
	}
	h.mu.Lock()
	if h.fetching[fileID] {
		h.mu.Unlock()
		if tr, ok := h.xfers.get(fileID); ok {
			return tr, nil
		}
		return Transfer{}, fmt.Errorf("chat: fetch already in progress")
	}
	h.fetching[fileID] = true
	h.mu.Unlock()

	h.xfers.put(Transfer{
		FileID:  fileID,
		Name:    msg.FileName,
		Total:   msg.Size,
		Percent: 0,
		Status:  TransferDownloading,
	})
	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.fetching, fileID)
			h.mu.Unlock()
		}()
		_, _ = h.fetchLocked(fileID, msg)
	}()
	tr, _ := h.xfers.get(fileID)
	return tr, nil
}

// CancelFetch aborts an in-flight download; partial file is discarded (P053).
func (h *Hub) CancelFetch(fileID string) (Transfer, error) {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return Transfer{}, fmt.Errorf("chat: empty file_id")
	}
	tr, ok := h.xfers.get(fileID)
	if !ok {
		return Transfer{}, fmt.Errorf("chat: unknown transfer")
	}
	if tr.Status == TransferDone {
		return tr, fmt.Errorf("chat: transfer already done")
	}
	if tr.Status == TransferCancelled {
		h.discardInbox(fileID, tr.Name)
		return tr, nil
	}

	h.mu.Lock()
	cancel := h.cancels[fileID]
	h.mu.Unlock()
	if cancel != nil {
		cancel()
	}

	// Mark cancelled immediately so UX never shows success after cancel.
	cancelled := Transfer{
		FileID:   fileID,
		Name:     tr.Name,
		Received: tr.Received,
		Total:    tr.Total,
		Percent:  tr.Percent,
		Status:   TransferCancelled,
		Error:    "discarded",
	}
	h.xfers.put(cancelled)

	// Wait for the fetch worker to unwind so .partial is removed.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		h.mu.Lock()
		still := h.fetching[fileID]
		h.mu.Unlock()
		if !still {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	h.discardInbox(fileID, tr.Name)
	h.logf("file_fetch_cancelled file_id=%s percent=%d", fileID, tr.Percent)
	if final, ok := h.xfers.get(fileID); ok {
		return final, nil
	}
	return cancelled, nil
}

func (h *Hub) discardInbox(fileID, name string) {
	if h.inboxDir == "" || fileID == "" {
		return
	}
	dest, err := files.InboxPath(h.inboxDir, fileID, name)
	if err != nil {
		return
	}
	_ = os.Remove(dest)
	_ = os.Remove(dest + ".partial")
}

// Fetch downloads file_id from the announce source and returns local path metadata.
func (h *Hub) Fetch(fileID string) (FetchResult, error) {
	msg, ok := h.store.FindFile(fileID)
	if !ok {
		return FetchResult{}, fmt.Errorf("chat: unknown file_id")
	}
	h.mu.Lock()
	if h.fetching[fileID] {
		h.mu.Unlock()
		return FetchResult{}, fmt.Errorf("chat: fetch already in progress")
	}
	h.fetching[fileID] = true
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.fetching, fileID)
		h.mu.Unlock()
	}()
	return h.fetchLocked(fileID, msg)
}

func (h *Hub) fetchLocked(fileID string, msg Message) (FetchResult, error) {
	ctx, cancel := context.WithCancel(context.Background())
	h.mu.Lock()
	h.cancels[fileID] = cancel
	h.mu.Unlock()
	defer func() {
		cancel()
		h.mu.Lock()
		delete(h.cancels, fileID)
		h.mu.Unlock()
	}()

	if tr, ok := h.xfers.get(fileID); !ok || tr.Status != TransferCancelled {
		h.xfers.put(Transfer{
			FileID:  fileID,
			Name:    msg.FileName,
			Total:   msg.Size,
			Percent: 0,
			Status:  TransferDownloading,
		})
	}

	update := func(recv int64) {
		ok := h.xfers.putActive(Transfer{
			FileID:   fileID,
			Name:     msg.FileName,
			Received: recv,
			Total:    msg.Size,
			Percent:  files.Percent(recv, msg.Size),
			Status:   TransferDownloading,
		})
		if !ok {
			return
		}
		if h.progressYield > 0 {
			time.Sleep(h.progressYield)
		} else {
			runtime.Gosched()
		}
	}

	if msg.PeerID == h.peerID {
		if err := ctx.Err(); err != nil {
			return h.finishCancelled(fileID, msg, 0, "")
		}
		if h.blobs != nil && h.blobs.Has(fileID) {
			sourcePath, err := h.blobs.Path(fileID)
			if err != nil {
				h.failTransfer(fileID, msg.FileName, msg.Size, err)
				return FetchResult{}, err
			}
			if err := files.VerifyFile(sourcePath, msg.Hash); err != nil {
				h.failTransfer(fileID, msg.FileName, msg.Size, err)
				return FetchResult{}, err
			}
			dest, err := files.InboxPath(h.inboxDir, fileID, msg.FileName)
			if err != nil {
				h.failTransfer(fileID, msg.FileName, msg.Size, err)
				return FetchResult{}, err
			}
			n, err := files.MaterializeLocal(ctx, sourcePath, dest, update)
			if errors.Is(err, files.ErrCancelled) || (err == nil && ctx.Err() != nil) {
				_ = os.Remove(dest)
				_ = os.Remove(dest + ".partial")
				return h.finishCancelled(fileID, msg, n, dest)
			}
			if err != nil {
				h.failTransfer(fileID, msg.FileName, msg.Size, err)
				return FetchResult{}, err
			}
			if err := files.VerifyFile(dest, msg.Hash); err != nil {
				_ = os.Remove(dest)
				h.failTransfer(fileID, msg.FileName, msg.Size, err)
				return FetchResult{}, err
			}
			if !h.xfers.putActive(Transfer{
				FileID:   fileID,
				Name:     msg.FileName,
				Received: n,
				Total:    msg.Size,
				Percent:  100,
				Status:   TransferDone,
				Path:     dest,
			}) {
				_ = os.Remove(dest)
				return h.finishCancelled(fileID, msg, n, dest)
			}
			h.logf("file_fetch_local_ok file_id=%s path=%s size=%d", fileID, dest, n)
			return FetchResult{
				FileID: fileID, Path: dest, Size: n, Name: msg.FileName,
				Percent: 100, Status: TransferDone,
			}, nil
		}
		err := fmt.Errorf("chat: local blob missing")
		h.failTransfer(fileID, msg.FileName, msg.Size, err)
		return FetchResult{}, err
	}

	var peer discovery.Peer
	found := false
	for _, p := range h.peers.List() {
		if p.PeerID == msg.PeerID {
			peer = p
			found = true
			break
		}
	}
	if !found || peer.Host == "" || peer.TCPPort <= 0 {
		err := fmt.Errorf("chat: source peer offline")
		h.failTransfer(fileID, msg.FileName, msg.Size, err)
		return FetchResult{}, err
	}
	if err := discovery.CheckDialHost(peer.Host); err != nil {
		h.failTransfer(fileID, msg.FileName, msg.Size, err)
		return FetchResult{}, err
	}

	dest, err := files.InboxPath(h.inboxDir, fileID, msg.FileName)
	if err != nil {
		h.failTransfer(fileID, msg.FileName, msg.Size, err)
		return FetchResult{}, err
	}

	addr := net.JoinHostPort(peer.Host, fmt.Sprintf("%d", peer.TCPPort))
	conn, err := h.dialer("tcp", addr, h.timeout)
	if err != nil {
		h.failTransfer(fileID, msg.FileName, msg.Size, err)
		return FetchResult{}, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	reqRaw, err := files.EncodeChunkReq(files.ChunkReq{
		FileID: fileID,
		Offset: 0,
		Limit:  h.chunkSize,
	})
	if err != nil {
		h.failTransfer(fileID, msg.FileName, msg.Size, err)
		return FetchResult{}, err
	}
	if _, err := conn.Write(reqRaw); err != nil {
		h.failTransfer(fileID, msg.FileName, msg.Size, err)
		return FetchResult{}, err
	}

	n, err := files.ReadChunks(ctx, conn, fileID, dest, msg.Size, func(recv, _ int64) {
		update(recv)
	})
	if errors.Is(err, files.ErrCancelled) || (err == nil && ctx.Err() != nil) {
		_ = os.Remove(dest)
		_ = os.Remove(dest + ".partial")
		return h.finishCancelled(fileID, msg, n, dest)
	}
	if err != nil {
		_ = os.Remove(dest)
		_ = os.Remove(dest + ".partial")
		if tr, ok := h.xfers.get(fileID); ok && tr.Status == TransferCancelled {
			return h.finishCancelled(fileID, msg, n, dest)
		}
		h.failTransfer(fileID, msg.FileName, msg.Size, err)
		return FetchResult{}, err
	}
	if err := files.VerifyFile(dest, msg.Hash); err != nil {
		_ = os.Remove(dest)
		_ = os.Remove(dest + ".partial")
		if tr, ok := h.xfers.get(fileID); ok && tr.Status == TransferCancelled {
			return h.finishCancelled(fileID, msg, n, dest)
		}
		h.failTransfer(fileID, msg.FileName, msg.Size, err)
		return FetchResult{}, err
	}
	if !h.xfers.putActive(Transfer{
		FileID:   fileID,
		Name:     msg.FileName,
		Received: n,
		Total:    msg.Size,
		Percent:  100,
		Status:   TransferDone,
		Path:     dest,
	}) {
		_ = os.Remove(dest)
		_ = os.Remove(dest + ".partial")
		return h.finishCancelled(fileID, msg, n, dest)
	}
	h.logf("file_fetch_ok file_id=%s path=%s size=%d", fileID, dest, n)
	return FetchResult{
		FileID: fileID, Path: dest, Size: n, Name: msg.FileName,
		Percent: 100, Status: TransferDone,
	}, nil
}

func (h *Hub) finishCancelled(fileID string, msg Message, recv int64, dest string) (FetchResult, error) {
	if dest != "" {
		_ = os.Remove(dest)
		_ = os.Remove(dest + ".partial")
	}
	tr := Transfer{
		FileID:   fileID,
		Name:     msg.FileName,
		Received: recv,
		Total:    msg.Size,
		Percent:  files.Percent(recv, msg.Size),
		Status:   TransferCancelled,
		Error:    "discarded",
	}
	if prev, ok := h.xfers.get(fileID); ok && prev.Status == TransferCancelled {
		tr.Percent = prev.Percent
		tr.Received = prev.Received
	}
	h.xfers.put(tr)
	return FetchResult{
		FileID: fileID, Name: msg.FileName, Size: 0,
		Percent: tr.Percent, Status: TransferCancelled,
	}, files.ErrCancelled
}

func (h *Hub) failTransfer(fileID, name string, total int64, err error) {
	if tr, ok := h.xfers.get(fileID); ok && tr.Status == TransferCancelled {
		return
	}
	h.xfers.put(Transfer{
		FileID:  fileID,
		Name:    name,
		Total:   total,
		Percent: files.Percent(0, total),
		Status:  TransferError,
		Error:   err.Error(),
	})
	h.logf("file_fetch_err file_id=%s err=%v", fileID, err)
}
