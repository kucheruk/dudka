package tui

import (
	"fmt"
	"strings"
)

// LargeFileBytes is the soft warning threshold (DUD-FILE-111 / P054): warn when size is greater.
const LargeFileBytes int64 = 100 * 1024 * 1024

// LargeFileWarningCopy is shown in TUI before starting a >100 MiB download.
const LargeFileWarningCopy = "ВНИМАНИЕ: файл больше 100 МиБ. Передача не запрещена — подтвердите: /fetch! <file_id>"

// IsLargeFile reports whether size should trigger a pre-start warning.
func IsLargeFile(size int64) bool {
	return size > LargeFileBytes
}

// FetchPlan is the outcome of BeginFetch (warning and/or started transfer).
type FetchPlan struct {
	FileID   string
	Name     string
	Size     int64
	Warning  string
	Started  bool
	Transfer TransferRow
}

// BeginFetch looks up announce size and either warns (large, !force) or starts the download.
// A warning does not hard-block: call again with force=true (or /fetch!) to proceed (P054).
func (c *Client) BeginFetch(fileID string, force bool) (FetchPlan, error) {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return FetchPlan{}, fmt.Errorf("tui: empty file_id")
	}
	snap, err := c.Fetch()
	if err != nil {
		return FetchPlan{}, err
	}
	var size int64
	var name string
	found := false
	for _, m := range snap.Messages {
		if m.FileID == fileID && (m.Type == MsgTypeFileAnnounce || m.FileName != "") {
			size = m.Size
			name = m.FileName
			found = true
			break
		}
	}
	if !found {
		return FetchPlan{}, fmt.Errorf("tui: unknown file_id %s", fileID)
	}
	plan := FetchPlan{FileID: fileID, Name: name, Size: size}
	if IsLargeFile(size) && !force {
		plan.Warning = LargeFileWarningCopy
		return plan, nil
	}
	tr, err := c.StartFetch(fileID)
	if err != nil {
		return plan, err
	}
	plan.Started = true
	plan.Transfer = tr
	return plan, nil
}
