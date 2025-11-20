package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bit2swaz/crisismesh/internal/protocol"
	"github.com/bit2swaz/crisismesh/internal/store"
	"github.com/bit2swaz/crisismesh/internal/transport"
)

// Helper to create a test node
func CreateTestNode(t *testing.T, nick string, port int) (*GossipEngine, string, func()) {
	// Use a temp file for DB to ensure isolation, or in-memory if supported by sqlite driver properly for concurrent access
	// Using file-based sqlite for robustness in tests
	dbPath := fmt.Sprintf("test_%s_%d.db", nick, time.Now().UnixNano())
	db, err := store.Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB for %s: %v", nick, err)
	}

	tm := transport.NewManager()
	nodeID := fmt.Sprintf("node-%s", nick)

	eng := NewGossipEngine(db, tm, nodeID, nick, port)

	ctx, cancel := context.WithCancel(context.Background())

	// Start engine components manually to avoid full Start() which includes UDP discovery
	// We only want TCP for this test
	if err := tm.Listen(fmt.Sprintf("%d", port), eng.handleConnection); err != nil {
		t.Fatalf("Failed to listen for %s: %v", nick, err)
	}

	// Start a periodic SYNC emitter for the test
	// In the real app, this might be triggered by events or a ticker.
	// For the test, we'll simulate it or rely on the fact that we need to trigger sync.
	// Since we haven't implemented periodic sync in Start() yet, let's add a simple ticker here
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Broadcast SYNC to all connected peers
				msgs, _ := store.GetMessages(db, 100)
				var ids []string
				for _, m := range msgs {
					ids = append(ids, m.ID)
				}
				syncPayload := protocol.SyncPayload{MessageIDs: ids}
				pBytes, _ := json.Marshal(syncPayload)
				packet := protocol.Packet{Type: protocol.TypeSync, Payload: pBytes}
				data, _ := json.Marshal(packet)
				tm.BroadcastPacket(data)
			}
		}
	}()

	cleanup := func() {
		cancel()
		tm.CloseAll()
		os.Remove(dbPath)
	}

	return eng, nodeID, cleanup
}

func TestGossipPropagation(t *testing.T) {
	// 1. Spawn Node A, B, C
	portA := 10001
	portB := 10002
	portC := 10003

	engA, _, cleanupA := CreateTestNode(t, "A", portA)
	defer cleanupA()

	engB, _, cleanupB := CreateTestNode(t, "B", portB)
	defer cleanupB()

	engC, _, cleanupC := CreateTestNode(t, "C", portC)
	defer cleanupC()

	// 2. Connect A->B and B->C
	// Wait a bit for listeners to start
	time.Sleep(100 * time.Millisecond)

	connAB, err := engA.transport.Dial(fmt.Sprintf("127.0.0.1:%d", portB))
	if err != nil {
		t.Fatalf("Failed to dial A->B: %v", err)
	}
	go engA.handleConnection(connAB)

	connBC, err := engB.transport.Dial(fmt.Sprintf("127.0.0.1:%d", portC))
	if err != nil {
		t.Fatalf("Failed to dial B->C: %v", err)
	}
	go engB.handleConnection(connBC)

	// 3. Inject message into Node A
	msgID := "msg-test-1"
	msg := &store.Message{
		ID:        msgID,
		SenderID:  engA.nodeID,
		Content:   "Gossip works!",
		Timestamp: time.Now().Unix(),
		Status:    "sent",
	}
	if err := store.SaveMessage(engA.db, msg); err != nil {
		t.Fatalf("Failed to save message to A: %v", err)
	}

	// 4. Wait for propagation
	// A has msg. A syncs to B. B requests msg. B gets msg.
	// B syncs to C. C requests msg. C gets msg.
	time.Sleep(3 * time.Second)

	// 5. Assert Node C has the message
	var retrievedMsg store.Message
	if err := engC.db.First(&retrievedMsg, "id = ?", msgID).Error; err != nil {
		t.Fatalf("Node C did not receive the message: %v", err)
	}

	if retrievedMsg.Content != "Gossip works!" {
		t.Errorf("Node C received wrong content: %s", retrievedMsg.Content)
	}
}
