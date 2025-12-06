package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/bit2swaz/crisismesh/internal/core"
	"github.com/bit2swaz/crisismesh/internal/discovery"
	"github.com/bit2swaz/crisismesh/internal/protocol"
	"github.com/bit2swaz/crisismesh/internal/store"
	"github.com/bit2swaz/crisismesh/internal/transport"
	"gorm.io/gorm"
)

type GossipEngine struct {
	db          *gorm.DB
	transport   *transport.Manager
	nodeID      string
	nick        string
	port        int
	peerChan    chan discovery.PeerInfo
	MsgUpdates  chan store.Message
	PeerUpdates chan []store.Peer
}

func NewGossipEngine(db *gorm.DB, tm *transport.Manager, nodeID, nick string, port int) *GossipEngine {
	return &GossipEngine{
		db:          db,
		transport:   tm,
		nodeID:      nodeID,
		nick:        nick,
		port:        port,
		peerChan:    make(chan discovery.PeerInfo, 10),
		MsgUpdates:  make(chan store.Message, 100),
		PeerUpdates: make(chan []store.Peer, 10),
	}
}

func (g *GossipEngine) GetNodeID() string {
	return g.nodeID
}

func (g *GossipEngine) Start(ctx context.Context) error {
	go func() {
		if err := discovery.StartHeartbeat(ctx, g.port, g.nodeID, g.nick); err != nil {
			slog.Error("Heartbeat failed", "error", err)
		}
	}()
	go func() {
		if err := discovery.StartListener(ctx, g.port, g.nodeID, g.peerChan); err != nil {
			slog.Error("Listener failed", "error", err)
		}
	}()
	if err := g.transport.Listen(fmt.Sprintf("%d", g.port), g.handleConnection); err != nil {
		return fmt.Errorf("failed to start TCP listener: %w", err)
	}
	go discovery.StartReaper(ctx, g.db)
	go g.startSyncer(ctx)
	go g.processPeers(ctx)
	return nil
}
func (g *GossipEngine) startSyncer(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			peers, err := store.GetActivePeers(g.db)
			if err != nil || len(peers) == 0 {
				continue
			}
			target := peers[rand.Intn(len(peers))]
			msgs, err := store.GetMessages(g.db, 50)
			if err != nil {
				slog.Error("Failed to get messages for sync", "error", err)
				continue
			}
			var ids []string
			for _, m := range msgs {
				ids = append(ids, m.ID)
			}
			if len(ids) == 0 {
				continue
			}
			syncPayload := protocol.SyncPayload{MessageIDs: ids}
			pBytes, err := json.Marshal(syncPayload)
			if err != nil {
				continue
			}
			packet := protocol.Packet{Type: protocol.TypeSync, Payload: pBytes}
			data, err := json.Marshal(packet)
			if err != nil {
				continue
			}
			if err := g.transport.SendPacket(target.Addr, data); err != nil {
				slog.Debug("Failed to gossip sync", "peer", target.Addr, "error", err)
			}
		}
	}
}
func (g *GossipEngine) processPeers(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case info := <-g.peerChan:
			g.handlePeerDiscovery(info)
		}
	}
}
func (g *GossipEngine) handlePeerDiscovery(info discovery.PeerInfo) {
	if g.port == 9001 && strings.Contains(info.Addr, "9003") {
		return
	}
	if g.port == 9003 && strings.Contains(info.Addr, "9001") {
		return
	}
	peer := store.Peer{
		ID:       info.ID,
		Nick:     info.Nick,
		Addr:     info.Addr,
		LastSeen: time.Now(),
		IsActive: true,
	}
	if err := store.UpsertPeer(g.db, peer); err != nil {
		slog.Error("Failed to upsert peer", "error", err)
	}
	var peers []store.Peer
	g.db.Find(&peers)
	select {
	case g.PeerUpdates <- peers:
	default:
	}
	if !g.transport.HasConnection(info.Addr) {
		slog.Info("Dialing peer", "addr", info.Addr)
		conn, err := g.transport.Dial(info.Addr)
		if err != nil {
			slog.Error("Failed to dial peer", "addr", info.Addr, "error", err)
			return
		}
		go g.handleConnection(conn)
	}
}
func (g *GossipEngine) handleConnection(conn net.Conn) {
	defer conn.Close()
	msgs, err := store.GetMessages(g.db, 50)
	if err == nil {
		var ids []string
		for _, m := range msgs {
			ids = append(ids, m.ID)
		}
		if len(ids) > 0 {
			slog.Info("Sending Initial SYNC", "count", len(ids), "remote", conn.RemoteAddr())
			syncPayload := protocol.SyncPayload{MessageIDs: ids}
			pBytes, _ := json.Marshal(syncPayload)
			packet := protocol.Packet{Type: protocol.TypeSync, Payload: pBytes}
			data, _ := json.Marshal(packet)
			transport.WriteFrame(conn, data)
		}
	}
	for {
		payload, err := transport.ReadFrame(conn)
		if err != nil {
			return
		}
		g.handlePacket(conn, payload)
	}
}
func (g *GossipEngine) PublishText(content string) error {
	ts := time.Now().Unix()
	msgID := core.GenerateMessageID(g.nodeID, content, ts)
	msg := store.Message{
		ID:        msgID,
		SenderID:  g.nodeID,
		Content:   content,
		Timestamp: ts,
		TTL:       10,
		HopCount:  0,
		Status:    "sent",
	}
	if err := store.SaveMessage(g.db, &msg); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	select {
	case g.MsgUpdates <- msg:
	default:
	}
	msgPayload := protocol.MsgPayload{Message: msg}
	pBytes, err := json.Marshal(msgPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal msg payload: %w", err)
	}
	packet := protocol.Packet{Type: protocol.TypeMsg, Payload: pBytes}
	data, err := json.Marshal(packet)
	if err != nil {
		return fmt.Errorf("failed to marshal packet: %w", err)
	}
	g.transport.BroadcastPacket(data)
	return nil
}
func (g *GossipEngine) ManualConnect(addr string) error {
	slog.Info("Manual connect initiated", "addr", addr)
	conn, err := g.transport.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to dial peer: %w", err)
	}
	go g.handleConnection(conn)
	return nil
}
func (g *GossipEngine) BroadcastSafe() error {
	content := "SAFE ALERT: I am safe!"
	ts := time.Now().Unix()
	msgID := core.GenerateMessageID(g.nodeID, content, ts)
	msg := store.Message{
		ID:        msgID,
		SenderID:  g.nodeID,
		Content:   content,
		Timestamp: ts,
		TTL:       10,
		HopCount:  0,
		Status:    "sent",
		Priority:  2,
	}
	if err := store.SaveMessage(g.db, &msg); err != nil {
		return fmt.Errorf("failed to save safe message: %w", err)
	}
	select {
	case g.MsgUpdates <- msg:
	default:
	}
	msgPayload := protocol.MsgPayload{Message: msg}
	pBytes, err := json.Marshal(msgPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal msg payload: %w", err)
	}
	packet := protocol.Packet{Type: protocol.TypeMsg, Payload: pBytes}
	data, err := json.Marshal(packet)
	if err != nil {
		return fmt.Errorf("failed to marshal packet: %w", err)
	}
	g.transport.BroadcastPacket(data)
	return nil
}
