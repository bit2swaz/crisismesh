package discovery

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestHeartbeatListener(t *testing.T) {
	port := 9999
	peerChan := make(chan PeerInfo, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Start Listener
	go func() {
		if err := StartListener(ctx, port, "my-node-id", peerChan); err != nil {
			// StartListener returns nil on context cancel, so real errors are failures
			t.Errorf("StartListener failed: %v", err)
		}
	}()

	// Give listener a moment to start
	time.Sleep(100 * time.Millisecond)

	// 2. Send valid heartbeat
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:9999")
	if err != nil {
		t.Fatalf("Failed to resolve addr: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		t.Fatalf("Failed to dial UDP: %v", err)
	}
	defer conn.Close()

	packet := HeartbeatPacket{
		Type: "beat",
		ID:   "peer-node-id",
		Nick: "PeerNick",
		Port: 12345,
		TS:   time.Now().Unix(),
	}
	data, _ := json.Marshal(packet)

	if _, err := conn.Write(data); err != nil {
		t.Fatalf("Failed to write packet: %v", err)
	}

	// 3. Assert reception
	select {
	case info := <-peerChan:
		if info.ID != "peer-node-id" {
			t.Errorf("Expected ID 'peer-node-id', got %q", info.ID)
		}
		if info.Nick != "PeerNick" {
			t.Errorf("Expected Nick 'PeerNick', got %q", info.Nick)
		}
		// Check Addr contains the port we sent
		expectedPort := "12345"
		if _, port, _ := net.SplitHostPort(info.Addr); port != expectedPort {
			t.Errorf("Expected Port %q, got %q in Addr %q", expectedPort, port, info.Addr)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for peer info")
	}

	// 4. Send malformed packet
	if _, err := conn.Write([]byte("{invalid-json")); err != nil {
		t.Fatalf("Failed to write malformed packet: %v", err)
	}

	// Ensure listener is still running by sending another valid packet
	packet.ID = "peer-node-id-2"
	data, _ = json.Marshal(packet)
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("Failed to write second packet: %v", err)
	}

	select {
	case info := <-peerChan:
		if info.ID != "peer-node-id-2" {
			t.Errorf("Expected ID 'peer-node-id-2', got %q", info.ID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for second peer info (listener might have crashed)")
	}
}
