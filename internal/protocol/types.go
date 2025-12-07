package protocol

import "github.com/bit2swaz/crisismesh/internal/store"

const (
	TypeSync = "SYNC"
	TypeReq  = "REQ"
	TypeMsg  = "MSG"
)

type Packet struct {
	Type    string `json:"type"`
	Payload []byte `json:"payload"`
}
type SyncPayload struct {
	MessageIDs []string `json:"message_ids"`
}
type ReqPayload struct {
	MessageIDs []string `json:"message_ids"`
}
type MsgPayload struct {
	Message store.Message `json:"message"`
}
