package chat

import "sync"

// Transfer lifecycle statuses (P052 / P053).
const (
	TransferDownloading = "downloading"
	TransferDone        = "done"
	TransferError       = "error"
	TransferCancelled   = "cancelled"
)

// Transfer is a download progress snapshot for loopback/TUI (0–100%).
type Transfer struct {
	FileID   string `json:"file_id"`
	Name     string `json:"name"`
	Received int64  `json:"received"`
	Total    int64  `json:"total"`
	Percent  int    `json:"percent"`
	Status   string `json:"status"`
	Path     string `json:"path,omitempty"`
	Error    string `json:"error,omitempty"`
}

type transferBook struct {
	mu   sync.RWMutex
	byID map[string]Transfer
}

func newTransferBook() *transferBook {
	return &transferBook{byID: make(map[string]Transfer)}
}

func (b *transferBook) list() []Transfer {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]Transfer, 0, len(b.byID))
	for _, t := range b.byID {
		out = append(out, t)
	}
	return out
}

func (b *transferBook) get(fileID string) (Transfer, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	t, ok := b.byID[fileID]
	return t, ok
}

func (b *transferBook) put(t Transfer) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.byID[t.FileID] = t
}

// putActive updates a transfer only while it is still downloadable (not cancelled/done).
func (b *transferBook) putActive(t Transfer) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if cur, ok := b.byID[t.FileID]; ok {
		if cur.Status == TransferCancelled || cur.Status == TransferDone {
			return false
		}
	}
	b.byID[t.FileID] = t
	return true
}
