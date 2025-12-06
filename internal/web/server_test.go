package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bit2swaz/crisismesh/internal/store"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// MockEngine implements the Engine interface for testing
type MockEngine struct {
	NodeID      string
	LastMessage string
}

func (m *MockEngine) GetNodeID() string {
	return m.NodeID
}

func (m *MockEngine) PublishText(content string) error {
	m.LastMessage = content
	return nil
}

func setupTestServer(t *testing.T) (*Server, *MockEngine, *gorm.DB) {
	// Setup in-memory DB
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}

	// AutoMigrate to ensure tables exist
	if err := db.AutoMigrate(&store.Message{}, &store.Peer{}); err != nil {
		t.Fatalf("Failed to migrate DB: %v", err)
	}

	mockEngine := &MockEngine{NodeID: "TEST_NODE_1"}
	server := NewServer(db, mockEngine, 8080)

	return server, mockEngine, db
}

func TestStaticAssets(t *testing.T) {
	server, _, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.handleIndex(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check for title in body
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	body := buf.String()

	if !strings.Contains(body, "<title>CrisisMesh Web Uplink</title>") {
		t.Errorf("Expected title 'CrisisMesh Web Uplink', got body: %s", body[:100])
	}
}

func TestAPIMessages(t *testing.T) {
	server, _, db := setupTestServer(t)

	// Insert dummy message
	msg := store.Message{
		ID:        "msg1",
		SenderID:  "peer1",
		Content:   "Hello World",
		Timestamp: time.Now().Unix(),
	}
	db.Create(&msg)

	req := httptest.NewRequest("GET", "/api/messages", nil)
	w := httptest.NewRecorder()

	server.handleMessages(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var messages []store.Message
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}
	if messages[0].Content != "Hello World" {
		t.Errorf("Expected content 'Hello World', got '%s'", messages[0].Content)
	}
}

func TestPostMessage(t *testing.T) {
	server, mockEngine, _ := setupTestServer(t)

	payload := map[string]interface{}{
		"content":  "Hello Web",
		"priority": 1,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/messages", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleMessages(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Assert Broadcaster triggered (MockEngine.PublishText called)
	if mockEngine.LastMessage != "Hello Web" {
		t.Errorf("Expected LastMessage 'Hello Web', got '%s'", mockEngine.LastMessage)
	}
}
