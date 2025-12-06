package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/bit2swaz/crisismesh/internal/store"
	"gorm.io/gorm"
)

type HeartbeatPacket struct {
	Type   string `json:"type"`
	ID     string `json:"id"`
	Nick   string `json:"nick"`
	Port   int    `json:"port"`
	TS     int64  `json:"ts"`
	PubKey string `json:"pub_key"`
}
type PeerInfo struct {
	ID     string
	Nick   string
	Addr   string
	PubKey string
}

func StartHeartbeat(ctx context.Context, servicePort int, nodeID, nick, pubKey string) error {
	targets := []string{"255.255.255.255", "127.0.0.1"}
	var conns []*net.UDPConn
	for _, host := range targets {
		for p := 9000; p <= 9005; p++ {
			addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, p))
			if err != nil {
				continue
			}
			conn, err := net.DialUDP("udp", nil, addr)
			if err == nil {
				conns = append(conns, conn)
			}
		}
	}
	if len(conns) == 0 {
		return fmt.Errorf("failed to dial any UDP broadcast addresses")
	}
	slog.Info("Heartbeat started", "targets", len(conns), "nodeID", nodeID)
	defer func() {
		for _, c := range conns {
			c.Close()
		}
	}()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case t := <-ticker.C:
			packet := HeartbeatPacket{
				Type:   "beat",
				ID:     nodeID,
				Nick:   nick,
				Port:   servicePort,
				TS:     t.Unix(),
				PubKey: pubKey,
			}
			data, err := json.Marshal(packet)
			if err != nil {
				continue
			}
			for _, c := range conns {
				_, _ = c.Write(data)
			}
		}
	}
}
func StartListener(ctx context.Context, port int, nodeID string, peerChan chan<- PeerInfo) error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to resolve listen address: %w", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %w", err)
	}
	go func() {
		<-ctx.Done()
		conn.Close()
	}()
	buf := make([]byte, 4096)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return fmt.Errorf("read error: %w", err)
			}
		}
		var packet HeartbeatPacket
		if err := json.Unmarshal(buf[:n], &packet); err != nil {
			slog.Warn("Failed to unmarshal heartbeat", "error", err)
			continue
		}
		if packet.Type != "beat" {
			continue
		}
		if packet.ID == nodeID {
			continue
		}
		remoteIP := remoteAddr.IP.String()
		peerAddr := fmt.Sprintf("%s:%d", remoteIP, packet.Port)
		slog.Info("Received heartbeat", "from", packet.Nick, "addr", peerAddr)
		select {
		case peerChan <- PeerInfo{
			ID:     packet.ID,
			Nick:   packet.Nick,
			Addr:   peerAddr,
			PubKey: packet.PubKey,
		}:
		case <-ctx.Done():
			return nil
		}
	}
}
func StartReaper(ctx context.Context, db *gorm.DB) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			threshold := time.Now().Add(-5 * time.Second)
			db.Model(&store.Peer{}).
				Where("is_active = ? AND last_seen < ?", true, threshold).
				Update("is_active", false)
		}
	}
}
