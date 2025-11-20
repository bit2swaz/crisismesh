package protocol

import "github.com/bit2swaz/crisismesh/internal/store"

// Packet types
const (
	TypeSync = "SYNC"
	TypeReq  = "REQ"
	TypeMsg  = "MSG"
	TypeSafe = "SAFE"
)

// Packet is the generic container for all protocol messages.
type Packet struct {
	Type    string `json:"type"`
	Payload []byte `json:"payload"`
}

// SyncPayload is sent to announce known messages (e.g., Vector Clock or Bloom Filter).
// For simplicity, we just send a list of Message IDs we have.
type SyncPayload struct {
	MessageIDs []string `json:"message_ids"`
}

// ReqPayload requests specific messages we are missing.
type ReqPayload struct {
	MessageIDs []string `json:"message_ids"`
}

// MsgPayload contains the actual message data.
type MsgPayload struct {
	Message store.Message `json:"message"`
}

// SafePayload is a broadcast message indicating safety status.
type SafePayload struct {
	SenderID  string `json:"sender_id"`
	Timestamp int64  `json:"timestamp"`
	Status    string `json:"status"` // e.g., "SAFE", "DANGER"
}
