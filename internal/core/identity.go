package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"golang.org/x/crypto/nacl/box"
)

type Identity struct {
	NodeID  string `json:"node_id"`
	PubKey  string `json:"pub_key"`
	PrivKey string `json:"priv_key"`
}

func GenerateIdentity() (*Identity, error) {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate keys: %w", err)
	}

	return &Identity{
		NodeID:  uuid.New().String(),
		PubKey:  hex.EncodeToString(pub[:]),
		PrivKey: hex.EncodeToString(priv[:]),
	}, nil
}

func LoadOrGenerateIdentity(filename string) (*Identity, error) {
	if _, err := os.Stat(filename); err == nil {
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read identity file: %w", err)
		}
		var id Identity
		if err := json.Unmarshal(data, &id); err != nil {
			return nil, fmt.Errorf("failed to parse identity file: %w", err)
		}
		if id.NodeID != "" && id.PubKey != "" && id.PrivKey != "" {
			return &id, nil
		}
	}

	id, err := GenerateIdentity()
	if err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(id, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal identity: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write identity file: %w", err)
	}

	return id, nil
}
func GenerateMessageID(senderID, content string, ts int64) string {
	input := fmt.Sprintf("%s:%s:%d", senderID, content, ts)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
