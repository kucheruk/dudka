package chat

import (
	"fmt"
	"net"
	"runtime"
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
	h.xfers.put(Transfer{
		FileID:  fileID,
		Name:    msg.FileName,
		Total:   msg.Size,
		Percent: 0,
		Status:  TransferDownloading,
	})

	update := func(recv int64) {
		h.xfers.put(Transfer{
			FileID:   fileID,
			Name:     msg.FileName,
			Received: recv,
			Total:    msg.Size,
			Percent:  files.Percent(recv, msg.Size),
			Status:   TransferDownloading,
		})
		if h.progressYield > 0 {
			time.Sleep(h.progressYield)
		} else {
			runtime.Gosched()
		}
	}

	if msg.PeerID == h.peerID {
		if h.blobs != nil && h.blobs.Has(fileID) {
			p, err := h.blobs.Path(fileID)
			if err != nil {
				h.failTransfer(fileID, msg.FileName, msg.Size, err)
				return FetchResult{}, err
			}
			h.xfers.put(Transfer{
				FileID:   fileID,
				Name:     msg.FileName,
				Received: msg.Size,
				Total:    msg.Size,
				Percent:  100,
				Status:   TransferDone,
				Path:     p,
			})
			return FetchResult{
				FileID: fileID, Path: p, Size: msg.Size, Name: msg.FileName,
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

	n, err := files.ReadChunks(conn, fileID, dest, msg.Size, func(recv, _ int64) {
		update(recv)
	})
	if err != nil {
		h.failTransfer(fileID, msg.FileName, msg.Size, err)
		return FetchResult{}, err
	}
	h.xfers.put(Transfer{
		FileID:   fileID,
		Name:     msg.FileName,
		Received: n,
		Total:    msg.Size,
		Percent:  100,
		Status:   TransferDone,
		Path:     dest,
	})
	h.logf("file_fetch_ok file_id=%s path=%s size=%d", fileID, dest, n)
	return FetchResult{
		FileID: fileID, Path: dest, Size: n, Name: msg.FileName,
		Percent: 100, Status: TransferDone,
	}, nil
}

func (h *Hub) failTransfer(fileID, name string, total int64, err error) {
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
