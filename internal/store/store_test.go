package store

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWALMode(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "wal_test.db")
	db, err := Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}

	var mode string
	if err := db.Raw("PRAGMA journal_mode;").Scan(&mode).Error; err != nil {
		t.Fatalf("Failed to query journal mode: %v", err)
	}

	if strings.ToLower(mode) != "wal" {
		t.Errorf("Expected journal_mode=wal, got %s", mode)
	}
}

func TestConcurrencyStress(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "stress_test.db")
	db, err := Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 50
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			msg := &Message{
				ID:          fmt.Sprintf("msg_%d", id),
				SenderID:    "stress_tester",
				RecipientID: "BROADCAST",
				Content:     fmt.Sprintf("Stress test message %d", id),
				Priority:    1,
				Timestamp:   time.Now().Unix(),
				TTL:         7,
				HopCount:    0,
				Status:      "pending",
			}
			if err := SaveMessage(db, msg); err != nil {
				errChan <- fmt.Errorf("goroutine %d failed: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("Concurrency error: %v", err)
	}

	msgs, err := GetMessages(db, 100)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}

	if len(msgs) != numGoroutines {
		t.Errorf("Expected %d messages, got %d", numGoroutines, len(msgs))
	}
}

func TestMessagePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	msg := &Message{
		ID:          "msg1",
		SenderID:    "sender1",
		RecipientID: "recipient1",
		Content:     "Hello World",
		Priority:    1,
		Timestamp:   time.Now().Unix(),
		TTL:         10,
		HopCount:    0,
		Status:      "sent",
	}
	if err := SaveMessage(db, msg); err != nil {
		t.Fatalf("Failed to save message: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying sql.DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("Failed to close db: %v", err)
	}
	db2, err := Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to re-open db: %v", err)
	}
	var retrievedMsg Message
	if err := db2.First(&retrievedMsg, "id = ?", "msg1").Error; err != nil {
		t.Fatalf("Failed to retrieve message: %v", err)
	}
	if retrievedMsg.Content != msg.Content {
		t.Errorf("Expected content %q, got %q", msg.Content, retrievedMsg.Content)
	}
	if retrievedMsg.SenderID != msg.SenderID {
		t.Errorf("Expected sender %q, got %q", msg.SenderID, retrievedMsg.SenderID)
	}
	peer := Peer{
		ID:       "peer1",
		Nick:     "Alice",
		Addr:     "127.0.0.1:9000",
		LastSeen: time.Now().Add(-1 * time.Hour),
		IsActive: true,
	}
	if err := UpsertPeer(db2, peer); err != nil {
		t.Fatalf("Failed to insert peer: %v", err)
	}
	newTime := time.Now()
	peer.LastSeen = newTime
	if err := UpsertPeer(db2, peer); err != nil {
		t.Fatalf("Failed to update peer: %v", err)
	}
	var retrievedPeer Peer
	if err := db2.First(&retrievedPeer, "id = ?", "peer1").Error; err != nil {
		t.Fatalf("Failed to retrieve peer: %v", err)
	}
	if !retrievedPeer.LastSeen.Equal(newTime) {
		diff := retrievedPeer.LastSeen.Sub(newTime)
		if diff < -time.Second || diff > time.Second {
			t.Errorf("Expected LastSeen to be updated to %v, got %v (diff: %v)", newTime, retrievedPeer.LastSeen, diff)
		}
	}
}
