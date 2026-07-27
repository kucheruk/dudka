package chat

import (
	"fmt"
	"net"
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
	FileID string `json:"file_id"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	Name   string `json:"name"`
}

// FetchFile downloads file_id from its announce source peer via chunks and writes inbox path.
func (h *Hub) FetchFile(fileID string) (string, error) {
	res, err := h.Fetch(fileID)
	if err != nil {
		return "", err
	}
	return res.Path, nil
}

// Fetch downloads file_id from the announce source and returns local path metadata.
func (h *Hub) Fetch(fileID string) (FetchResult, error) {
	msg, ok := h.store.FindFile(fileID)
	if !ok {
		return FetchResult{}, fmt.Errorf("chat: unknown file_id")
	}
	if msg.PeerID == h.peerID {
		if h.blobs != nil && h.blobs.Has(fileID) {
			p, err := h.blobs.Path(fileID)
			if err != nil {
				return FetchResult{}, err
			}
			return FetchResult{FileID: fileID, Path: p, Size: msg.Size, Name: msg.FileName}, nil
		}
		return FetchResult{}, fmt.Errorf("chat: local blob missing")
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
		return FetchResult{}, fmt.Errorf("chat: source peer offline")
	}
	if err := discovery.CheckDialHost(peer.Host); err != nil {
		return FetchResult{}, err
	}

	dest, err := files.InboxPath(h.inboxDir, fileID, msg.FileName)
	if err != nil {
		return FetchResult{}, err
	}

	addr := net.JoinHostPort(peer.Host, fmt.Sprintf("%d", peer.TCPPort))
	conn, err := h.dialer("tcp", addr, h.timeout)
	if err != nil {
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
		return FetchResult{}, err
	}
	if _, err := conn.Write(reqRaw); err != nil {
		return FetchResult{}, err
	}

	n, err := files.ReadChunks(conn, fileID, dest)
	if err != nil {
		return FetchResult{}, err
	}
	h.logf("file_fetch_ok file_id=%s path=%s size=%d", fileID, dest, n)
	return FetchResult{FileID: fileID, Path: dest, Size: n, Name: msg.FileName}, nil
}
