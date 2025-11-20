package engine

import (
	"encoding/json"
	"log/slog"
	"net"

	"github.com/bit2swaz/crisismesh/internal/protocol"
	"github.com/bit2swaz/crisismesh/internal/store"
	"github.com/bit2swaz/crisismesh/internal/transport"
)

func (g *GossipEngine) handlePacket(conn net.Conn, data []byte) {
	var packet protocol.Packet
	if err := json.Unmarshal(data, &packet); err != nil {
		slog.Error("Failed to unmarshal packet", "error", err)
		return
	}

	switch packet.Type {
	case protocol.TypeSafe:
		g.handleSafe(packet.Payload)
	case protocol.TypeMsg:
		g.handleMsg(packet.Payload)
	case protocol.TypeSync:
		g.handleSync(conn, packet.Payload)
	case protocol.TypeReq:
		g.handleReq(conn, packet.Payload)
	default:
		slog.Warn("Unknown packet type", "type", packet.Type)
	}
}

func (g *GossipEngine) handleSafe(payload []byte) {
	var safe protocol.SafePayload
	if err := json.Unmarshal(payload, &safe); err != nil {
		slog.Error("Failed to unmarshal SAFE payload", "error", err)
		return
	}

	slog.Info("SAFE ALERT RECEIVED", "sender", safe.SenderID, "status", safe.Status)
	// TODO: Hook to UI
	// TODO: Save to DB if we track safety status history
	// TODO: Forwarding logic (requires TTL in SafePayload or wrapping Packet)
}

func (g *GossipEngine) handleMsg(payload []byte) {
	var msgPayload protocol.MsgPayload
	if err := json.Unmarshal(payload, &msgPayload); err != nil {
		slog.Error("Failed to unmarshal MSG payload", "error", err)
		return
	}

	msg := msgPayload.Message

	// Dedup check: Try to save. If it exists (primary key violation), GORM might return error or we can check first.
	// store.SaveMessage uses db.Create which returns error on duplicate PK.
	if err := store.SaveMessage(g.db, &msg); err != nil {
		// Assuming error means duplicate or DB issue.
		// If duplicate, we just ignore.
		// slog.Debug("Message already exists or failed to save", "id", msg.ID, "error", err)
		return
	}

	slog.Info("New message received", "id", msg.ID, "content", msg.Content)
}

func (g *GossipEngine) handleSync(conn net.Conn, payload []byte) {
	var sync protocol.SyncPayload
	if err := json.Unmarshal(payload, &sync); err != nil {
		slog.Error("Failed to unmarshal SYNC payload", "error", err)
		return
	}

	// Compare inventories
	// 1. Get all our message IDs
	// Optimization: In a real app, we wouldn't fetch all messages, but use Merkle Trees or Bloom Filters.
	// For now, let's fetch the last 100 messages to compare.
	msgs, err := store.GetMessages(g.db, 100)
	if err != nil {
		slog.Error("Failed to get local messages", "error", err)
		return
	}

	myIDs := make(map[string]bool)
	for _, m := range msgs {
		myIDs[m.ID] = true
	}

	// 2. Calculate missing IDs (what they have that we don't is not what we do here.
	// SYNC usually means "Here is what I have".
	// If they sent us their list, we check if WE are missing anything from THEIR list.
	var missingIDs []string
	for _, id := range sync.MessageIDs {
		if !myIDs[id] {
			missingIDs = append(missingIDs, id)
		}
	}

	if len(missingIDs) > 0 {
		// Send REQ
		req := protocol.ReqPayload{MessageIDs: missingIDs}
		reqBytes, _ := json.Marshal(req)
		packet := protocol.Packet{Type: protocol.TypeReq, Payload: reqBytes}
		data, _ := json.Marshal(packet)
		transport.WriteFrame(conn, data)
	}
}

func (g *GossipEngine) handleReq(conn net.Conn, payload []byte) {
	var req protocol.ReqPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		slog.Error("Failed to unmarshal REQ payload", "error", err)
		return
	}

	for _, id := range req.MessageIDs {
		var msg store.Message
		if err := g.db.First(&msg, "id = ?", id).Error; err == nil {
			// Found it, send it
			msgPayload := protocol.MsgPayload{Message: msg}
			pBytes, _ := json.Marshal(msgPayload)
			packet := protocol.Packet{Type: protocol.TypeMsg, Payload: pBytes}
			data, _ := json.Marshal(packet)
			transport.WriteFrame(conn, data)
		}
	}
}
