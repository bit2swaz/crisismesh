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
func CreateTestNode(t *testing.T, nick string, port int) (*GossipEngine, string, func()) {
	dbPath := fmt.Sprintf("test_%s_%d.db", nick, time.Now().UnixNano())
	db, err := store.Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB for %s: %v", nick, err)
	}
	tm := transport.NewManager()
	nodeID := fmt.Sprintf("node-%s", nick)
	eng := NewGossipEngine(db, tm, nodeID, nick, port)
	ctx, cancel := context.WithCancel(context.Background())
	if err := tm.Listen(fmt.Sprintf("%d", port), eng.handleConnection); err != nil {
		t.Fatalf("Failed to listen for %s: %v", nick, err)
	}
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
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
	portA := 10001
	portB := 10002
	portC := 10003
	engA, _, cleanupA := CreateTestNode(t, "A", portA)
	defer cleanupA()
	engB, _, cleanupB := CreateTestNode(t, "B", portB)
	defer cleanupB()
	engC, _, cleanupC := CreateTestNode(t, "C", portC)
	defer cleanupC()
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
	time.Sleep(3 * time.Second)
	var retrievedMsg store.Message
	if err := engC.db.First(&retrievedMsg, "id = ?", msgID).Error; err != nil {
		t.Fatalf("Node C did not receive the message: %v", err)
	}
	if retrievedMsg.Content != "Gossip works!" {
		t.Errorf("Node C received wrong content: %s", retrievedMsg.Content)
	}
}
