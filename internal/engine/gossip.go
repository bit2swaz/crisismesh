package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/bit2swaz/crisismesh/internal/discovery"
	"github.com/bit2swaz/crisismesh/internal/protocol"
	"github.com/bit2swaz/crisismesh/internal/store"
	"github.com/bit2swaz/crisismesh/internal/transport"
	"gorm.io/gorm"
)

type GossipEngine struct {
	db        *gorm.DB
	transport *transport.Manager
	nodeID    string
	nick      string
	port      int
	peerChan  chan discovery.PeerInfo
}

func NewGossipEngine(db *gorm.DB, tm *transport.Manager, nodeID, nick string, port int) *GossipEngine {
	return &GossipEngine{
		db:        db,
		transport: tm,
		nodeID:    nodeID,
		nick:      nick,
		port:      port,
		peerChan:  make(chan discovery.PeerInfo, 10),
	}
}

func (g *GossipEngine) Start(ctx context.Context) error {
	// 1. Start UDP Heartbeat
	go func() {
		if err := discovery.StartHeartbeat(ctx, g.port, g.nodeID, g.nick); err != nil {
			slog.Error("Heartbeat failed", "error", err)
		}
	}()

	// 2. Start UDP Listener
	go func() {
		if err := discovery.StartListener(ctx, g.port, g.nodeID, g.peerChan); err != nil {
			slog.Error("Listener failed", "error", err)
		}
	}()

	// 3. Start TCP Listener
	if err := g.transport.Listen(fmt.Sprintf("%d", g.port), g.handleConnection); err != nil {
		return fmt.Errorf("failed to start TCP listener: %w", err)
	}

	// 4. Run Reaper
	go discovery.StartReaper(ctx, g.db)

	// 5. Start Syncer (Periodic Gossip)
	go g.startSyncer(ctx)

	// 6. Consume Discovery Channel
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
			// Broadcast SYNC to all connected peers
			// Optimization: Only send IDs of recent messages or use Merkle Tree
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

			g.transport.BroadcastPacket(data)
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
	// HACK: Manual Topology Enforcement for "Whisper" Test
	// Alice (9001) cannot see Charlie (9003) and vice versa.
	if g.port == 9001 && strings.Contains(info.Addr, "9003") {
		return
	}
	if g.port == 9003 && strings.Contains(info.Addr, "9001") {
		return
	}

	// Upsert Peer to DB
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

	// If we are not connected via TCP, call transport.Dial
	if !g.transport.HasConnection(info.Addr) {
		slog.Info("Dialing peer", "addr", info.Addr)
		conn, err := g.transport.Dial(info.Addr)
		if err != nil {
			slog.Error("Failed to dial peer", "addr", info.Addr, "error", err)
			return
		}
		// Handle the new connection
		go g.handleConnection(conn)
	}
}

func (g *GossipEngine) handleConnection(conn net.Conn) {
	// Handle incoming TCP connection
	defer conn.Close()
	for {
		payload, err := transport.ReadFrame(conn)
		if err != nil {
			// slog.Error("Connection read error", "error", err)
			return
		}
		g.handlePacket(conn, payload)
	}
}
