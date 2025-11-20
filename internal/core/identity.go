package core
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"github.com/google/uuid"
)
type Identity struct {
	NodeID string `json:"node_id"`
}
func GenerateNodeID(filename string) (string, error) {
	if _, err := os.Stat(filename); err == nil {
		data, err := os.ReadFile(filename)
		if err != nil {
			return "", fmt.Errorf("failed to read identity file: %w", err)
		}
		var id Identity
		if err := json.Unmarshal(data, &id); err != nil {
			return "", fmt.Errorf("failed to parse identity file: %w", err)
		}
		if id.NodeID != "" {
			return id.NodeID, nil
		}
	}
	newID := uuid.New().String()
	id := Identity{NodeID: newID}
	data, err := json.MarshalIndent(id, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal identity: %w", err)
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write identity file: %w", err)
	}
	return newID, nil
}
func GenerateMessageID(senderID, content string, ts int64) string {
	input := fmt.Sprintf("%s:%s:%d", senderID, content, ts)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
