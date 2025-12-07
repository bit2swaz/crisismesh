package engine

import (
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net"

	"github.com/bit2swaz/crisismesh/internal/protocol"
	"github.com/bit2swaz/crisismesh/internal/store"
	"github.com/bit2swaz/crisismesh/internal/transport"
	"golang.org/x/crypto/nacl/box"
)

func (g *GossipEngine) handlePacket(conn net.Conn, data []byte) {
	var packet protocol.Packet
	if err := json.Unmarshal(data, &packet); err != nil {
		slog.Error("Failed to unmarshal packet", "error", err)
		return
	}
	switch packet.Type {
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
func (g *GossipEngine) handleMsg(payload []byte) {
	var msgPayload protocol.MsgPayload
	if err := json.Unmarshal(payload, &msgPayload); err != nil {
		slog.Error("Failed to unmarshal MSG payload", "error", err)
		return
	}
	msg := msgPayload.Message

	if msg.IsEncrypted && msg.RecipientID == g.nodeID {
		privKey, _ := hex.DecodeString(g.privKey)
		pubKey, _ := hex.DecodeString(g.pubKey)
		var pubKeyArr, privKeyArr [32]byte
		copy(pubKeyArr[:], pubKey)
		copy(privKeyArr[:], privKey)

		encrypted, _ := hex.DecodeString(msg.Content)
		decrypted, ok := box.OpenAnonymous(nil, encrypted, &pubKeyArr, &privKeyArr)
		if ok {
			msg.Content = string(decrypted)
			msg.IsEncrypted = false
		} else {
			slog.Error("Failed to decrypt message", "id", msg.ID)
		}
	}

	if err := store.SaveMessage(g.db, &msg); err != nil {
		return
	}
	slog.Info("New message received", "id", msg.ID, "content", msg.Content)
	select {
	case g.MsgUpdates <- msg:
	default:
	}

	// Send to Uplink if configured
	if g.UplinkChan != nil {
		select {
		case g.UplinkChan <- msg:
		default:
		}
	}
}
func (g *GossipEngine) handleSync(conn net.Conn, payload []byte) {
	var sync protocol.SyncPayload
	if err := json.Unmarshal(payload, &sync); err != nil {
		slog.Error("Failed to unmarshal SYNC payload", "error", err)
		return
	}
	slog.Info("Received SYNC", "count", len(sync.MessageIDs), "remote", conn.RemoteAddr())
	msgs, err := store.GetMessages(g.db, 100)
	if err != nil {
		slog.Error("Failed to get local messages", "error", err)
		return
	}
	myIDs := make(map[string]bool)
	for _, m := range msgs {
		myIDs[m.ID] = true
	}
	var missingIDs []string
	for _, id := range sync.MessageIDs {
		if !myIDs[id] {
			missingIDs = append(missingIDs, id)
		}
	}
	if len(missingIDs) > 0 {
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
			msgPayload := protocol.MsgPayload{Message: msg}
			pBytes, _ := json.Marshal(msgPayload)
			packet := protocol.Packet{Type: protocol.TypeMsg, Payload: pBytes}
			data, _ := json.Marshal(packet)
			transport.WriteFrame(conn, data)
		}
	}
}
