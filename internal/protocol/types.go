package protocol
import "github.com/bit2swaz/crisismesh/internal/store"
const (
	TypeSync = "SYNC"
	TypeReq  = "REQ"
	TypeMsg  = "MSG"
	TypeSafe = "SAFE"
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
type SafePayload struct {
	SenderID  string `json:"sender_id"`
	Timestamp int64  `json:"timestamp"`
	Status    string `json:"status"`  
}
