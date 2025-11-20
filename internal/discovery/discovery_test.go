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
	go func() {
		if err := StartListener(ctx, port, "my-node-id", peerChan); err != nil {
			t.Errorf("StartListener failed: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond)
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
	select {
	case info := <-peerChan:
		if info.ID != "peer-node-id" {
			t.Errorf("Expected ID 'peer-node-id', got %q", info.ID)
		}
		if info.Nick != "PeerNick" {
			t.Errorf("Expected Nick 'PeerNick', got %q", info.Nick)
		}
		expectedPort := "12345"
		if _, port, _ := net.SplitHostPort(info.Addr); port != expectedPort {
			t.Errorf("Expected Port %q, got %q in Addr %q", expectedPort, port, info.Addr)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for peer info")
	}
	if _, err := conn.Write([]byte("{invalid-json")); err != nil {
		t.Fatalf("Failed to write malformed packet: %v", err)
	}
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
