package chat

// Best-effort send statuses (DUD-CHAT-130 / P035). Never "delivered".
const (
	StatusAccepted = "accepted" // stored locally; no peers to fan out to
	StatusQueued   = "queued"   // stored locally; fan-out scheduled (not acked)
)

// SendResult is the POST /send response shape — no delivery guarantees.
type SendResult struct {
	Status  string  `json:"status"`
	Queued  int     `json:"queued"`
	Message Message `json:"message"`
}

// IsBestEffortStatus reports whether status is an allowed send outcome.
func IsBestEffortStatus(status string) bool {
	return status == StatusAccepted || status == StatusQueued
}

// SendStatusForQueued picks accepted vs queued from fan-out peer count.
func SendStatusForQueued(queued int) string {
	if queued > 0 {
		return StatusQueued
	}
	return StatusAccepted
}
